package auth

import (
	"context"
	"errors"
	"fmt"
	"gearr/helper"
	"gearr/model"
	"gearr/server/repository"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrOIDCNotEnabled      = errors.New("oidc not enabled")
	ErrAPITokensNotEnabled = errors.New("api tokens not enabled")
)

type AuthService struct {
	config       AuthConfig
	repo         repository.Repository
	oidcProvider *oidc.Provider
	oauth2Config *oauth2.Config
}

func NewAuthService(config AuthConfig, repo repository.Repository) (*AuthService, error) {
	service := &AuthService{
		config: config,
		repo:   repo,
	}

	if config.OIDC.Enabled {
		if err := service.initOIDC(); err != nil {
			return nil, fmt.Errorf("failed to initialize OIDC: %w", err)
		}
		helper.Info("OIDC authentication initialized")
	}

	if config.Session.Secret == "" {
		secret, err := GenerateToken()
		if err != nil {
			return nil, fmt.Errorf("failed to generate session secret: %w", err)
		}
		config.Session.Secret = secret
		service.config = config
	}

	return service, nil
}

func (s *AuthService) initOIDC() error {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, s.config.OIDC.Issuer)
	if err != nil {
		return fmt.Errorf("failed to create OIDC provider: %w", err)
	}
	s.oidcProvider = provider

	s.oauth2Config = &oauth2.Config{
		ClientID:     s.config.OIDC.ClientID,
		ClientSecret: s.config.OIDC.ClientSecret,
		RedirectURL:  s.config.OIDC.RedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       s.config.OIDC.Scopes,
	}

	return nil
}

func (s *AuthService) GetOAuth2Config() *oauth2.Config {
	return s.oauth2Config
}

func (s *AuthService) IsOIDCEnabled() bool {
	return s.config.OIDC.Enabled
}

func (s *AuthService) IsAPITokensEnabled() bool {
	return s.config.APITokens.Enabled
}

func (s *AuthService) GetStaticToken() string {
	return s.config.Token
}

func (s *AuthService) SetStaticToken(token string) {
	s.config.Token = token
}

func (s *AuthService) ValidateAPIToken(ctx context.Context, token string) (*repository.APIToken, error) {
	if !s.config.APITokens.Enabled {
		return nil, ErrAPITokensNotEnabled
	}

	tokenHash := HashToken(token)
	apiToken, err := s.repo.GetAPITokenByToken(ctx, tokenHash)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if apiToken.ExpiresAt != nil && apiToken.ExpiresAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	now := time.Now()
	apiToken.LastUsed = &now
	if err := s.repo.UpdateAPITokenLastUsed(ctx, apiToken.ID, now); err != nil {
		helper.Warnf("failed to update token last used: %s", err)
	}

	return apiToken, nil
}

func (s *AuthService) CreateAPIToken(ctx context.Context, name string, scope model.TokenScope, createdBy string, expiresAt *time.Time) (*repository.APIToken, string, error) {
	if !s.config.APITokens.Enabled {
		return nil, "", ErrAPITokensNotEnabled
	}

	if !model.IsValidScope(scope) {
		return nil, "", fmt.Errorf("invalid scope: %s", scope)
	}

	id, err := GenerateTokenID()
	if err != nil {
		return nil, "", err
	}

	token, err := GenerateToken()
	if err != nil {
		return nil, "", err
	}

	apiToken := &repository.APIToken{
		ID:        id,
		Name:      name,
		TokenHash: HashToken(token),
		Scope:     scope,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		CreatedBy: createdBy,
	}

	if err := s.repo.CreateAPIToken(ctx, apiToken); err != nil {
		return nil, "", err
	}

	return apiToken, token, nil
}

func (s *AuthService) ListAPITokens(ctx context.Context) ([]*repository.APIToken, error) {
	if !s.config.APITokens.Enabled {
		return nil, ErrAPITokensNotEnabled
	}
	return s.repo.ListAPITokens(ctx)
}

func (s *AuthService) DeleteAPIToken(ctx context.Context, id string) error {
	if !s.config.APITokens.Enabled {
		return ErrAPITokensNotEnabled
	}
	return s.repo.DeleteAPIToken(ctx, id)
}

func (s *AuthService) VerifyOIDCToken(ctx context.Context, rawToken string) (*oidc.IDToken, error) {
	if !s.config.OIDC.Enabled {
		return nil, ErrOIDCNotEnabled
	}

	verifier := s.oidcProvider.Verifier(&oidc.Config{ClientID: s.config.OIDC.ClientID})
	return verifier.Verify(ctx, rawToken)
}

type SessionClaims struct {
	UserID string           `json:"user_id"`
	Email  string           `json:"email"`
	Name   string           `json:"name"`
	Scope  model.TokenScope `json:"scope"`
	jwt.RegisteredClaims
}

func (s *AuthService) CreateSession(userID, email, name string) (string, error) {
	claims := SessionClaims{
		UserID: userID,
		Email:  email,
		Name:   name,
		Scope:  model.ScopeAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.Session.MaxAge)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Session.Secret))
}

func (s *AuthService) ValidateSession(tokenString string) (*SessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Session.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*SessionClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) ValidateBearerToken(ctx context.Context, authHeader string) (*repository.APIToken, *SessionClaims, error) {
	if authHeader == "" {
		return nil, nil, ErrUnauthorized
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return nil, nil, ErrUnauthorized
	}

	token := authHeader[len(bearerPrefix):]

	apiToken, err := s.ValidateAPIToken(ctx, token)
	if err == nil {
		return apiToken, nil, nil
	}

	if s.config.OIDC.Enabled {
		session, err := s.ValidateSession(token)
		if err == nil {
			return nil, session, nil
		}
	}

	if s.config.Token != "" && token == s.config.Token {
		return &repository.APIToken{
			ID:    "static",
			Name:  "static-token",
			Scope: model.ScopeAdmin,
		}, nil, nil
	}

	return nil, nil, ErrInvalidToken
}

type OIDCUserInfo struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

func (s *AuthService) GetUserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*OIDCUserInfo, error) {
	if !s.config.OIDC.Enabled {
		return nil, ErrOIDCNotEnabled
	}

	info, err := s.oidcProvider.UserInfo(ctx, tokenSource)
	if err != nil {
		return nil, err
	}

	var userInfo OIDCUserInfo
	if err := info.Claims(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (s *AuthService) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if !s.config.OIDC.Enabled {
		return nil, ErrOIDCNotEnabled
	}
	return s.oauth2Config.Exchange(ctx, code)
}

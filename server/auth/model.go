package auth

import (
	"crypto/rand"
	"encoding/hex"
	"gearr/model"
	"time"
)

type OIDCConfig struct {
	Enabled      bool     `mapstructure:"enabled" json:"enabled"`
	Issuer       string   `mapstructure:"issuer" json:"issuer"`
	ClientID     string   `mapstructure:"clientId" json:"client_id"`
	ClientSecret string   `mapstructure:"clientSecret" json:"-"`
	RedirectURI  string   `mapstructure:"redirectUri" json:"redirect_uri"`
	Scopes       []string `mapstructure:"scopes" json:"scopes"`
}

type APITokenConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled"`
}

type AuthConfig struct {
	OIDC      OIDCConfig     `mapstructure:"oidc" json:"oidc"`
	APITokens APITokenConfig `mapstructure:"apiTokens" json:"api_tokens"`
	Session   SessionConfig  `mapstructure:"session" json:"session"`
	Token     string         `mapstructure:"token" json:"-"`
}

type SessionConfig struct {
	Secret     string        `mapstructure:"secret" json:"-"`
	MaxAge     time.Duration `mapstructure:"maxAge" json:"max_age"`
	CookieName string        `mapstructure:"cookieName" json:"cookie_name"`
}

func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		OIDC: OIDCConfig{
			Enabled: false,
			Scopes:  []string{"openid", "profile", "email"},
		},
		APITokens: APITokenConfig{
			Enabled: true,
		},
		Session: SessionConfig{
			CookieName: "gearr_session",
			MaxAge:     time.Hour * 24,
		},
	}
}

func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateTokenID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func HasScope(required model.TokenScope, actual model.TokenScope) bool {
	return model.HasScope(required, actual)
}

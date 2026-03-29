package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gearr/helper"
	"gearr/internal/constants"
	"gearr/model"
	"gearr/server/auth"
	"gearr/server/repository"
	"gearr/server/scanner"
	"gearr/server/scheduler"
	"gearr/server/watcher"
	"gearr/server/web/ui"
	"gearr/server/webhook"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
)

type WebServer struct {
	WebServerConfig
	scheduler      scheduler.Scheduler
	scanner        *scanner.Scanner
	router         *gin.Engine
	ctx            context.Context
	upgrader       websocket.Upgrader
	watcherHandler *watcher.Handler
	webhookHandler *webhook.HTTPHandler
	webhookConfig  *model.WebhookConfig
	repo           repository.Repository
	authService    *auth.AuthService
}

func (w *WebServer) addJob(c *gin.Context) {
	var jobRequest model.JobRequest
	if err := c.ShouldBindJSON(&jobRequest); err != nil {
		webError(c, err, 500)
		return
	}

	job, err := w.scheduler.ScheduleJobRequest(w.ctx, &jobRequest)
	if err != nil {
		if errors.Is(err, model.ErrJobExists) {
			c.Status(http.StatusConflict)
			return
		}
		c.AbortWithStatusJSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (w *WebServer) getJobs(c *gin.Context) {
	jobs, err := w.scheduler.GetJobs(w.ctx)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func (w *WebServer) getJobByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("job ID parameter not found"), 404)
		return
	}

	job, err := w.scheduler.GetJob(w.ctx, id)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, job)
}

func (w *WebServer) deleteJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("job ID parameter not found"), 404)
		return
	}

	err := w.scheduler.DeleteJob(w.ctx, id)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (w *WebServer) updateJobPriority(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("job ID parameter not found"), 404)
		return
	}

	var req struct {
		Priority int `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if !model.JobPriorityIsValid(req.Priority) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "priority must be between 0 and 3"})
		return
	}

	err := w.scheduler.UpdateJobPriority(w.ctx, id, req.Priority)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "priority": req.Priority})
}

func (w *WebServer) getJobsUpdates(c *gin.Context) {
	conn, err := w.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		helper.Errorf("failed to upgrade: %s", err)
		return
	}
	defer conn.Close()
	helper.Debug("websocket connected")

	id, ch := w.scheduler.GetUpdateJobsChan(w.ctx)
	helper.Debug("channel connected")
	defer w.scheduler.CloseUpdateJobsChan(id)
	for {
		jobUpdateNotification, ok := <-ch
		if !ok {
			break
		}
		jsonBytes, err := json.Marshal(jobUpdateNotification)
		if err != nil {
			helper.Errorf("task cannot be marshal to json: %s", err)
			return
		}
		helper.Debugf("sending update: %+v", jobUpdateNotification)
		conn.WriteMessage(websocket.TextMessage, []byte(jsonBytes))
	}
}

func (w *WebServer) upload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("job ID parameter not found"), 404)
		return
	}

	uploadStream, err := w.scheduler.GetUploadJobWriter(c.Request.Context(), id)
	if errors.Is(err, scheduler.ErrorStreamNotAllowed) {
		webError(c, err, 403)
		return
	} else if errors.Is(err, scheduler.ErrorJobNotFound) {
		webError(c, err, 404)
		return
	} else if webError(c, err, 500) {
		return
	}
	defer uploadStream.Close(false)

	size, _ := strconv.ParseUint(c.GetHeader("Content-Length"), 10, 64)
	checksum := c.GetHeader("checksum")
	if checksum == "" {
		webError(c, fmt.Errorf("checksum is mandatory in the headers"), 403)
		return
	}

	b := make([]byte, constants.IOBufferSize)
	reader := c.Request.Body
	var readed uint64
loop:
	for {
		select {
		case <-c.Request.Context().Done():
			return
		default:
			readedBytes, err := reader.Read(b)
			readed += uint64(readedBytes)
			uploadStream.Write(b[:readedBytes])
			//TODO check error here?
			if err == io.EOF {
				break loop
			}
		}
	}
	if size != readed {
		defer uploadStream.Clean()
		webError(c, fmt.Errorf("invalid size, expected %d, received %d", size, readed), 400)
		return
	}
	checksumUpload := uploadStream.GetHash()
	if checksumUpload != checksum {
		defer uploadStream.Clean()
		webError(c, fmt.Errorf("invalid checksum, received %s, calculated %s", checksum, checksumUpload), 400)
		return
	}
	c.Status(http.StatusCreated)
}

func (w *WebServer) download(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("job ID parameter not found"), 404)
		return
	}

	downloadStream, err := w.scheduler.GetDownloadJobWriter(c.Request.Context(), id)
	if errors.Is(err, scheduler.ErrorStreamNotAllowed) {
		webError(c, err, 403)
		return
	} else if errors.Is(err, scheduler.ErrorJobNotFound) {
		webError(c, err, 404)
		return
	} else if webError(c, err, 500) {
		return
	}
	defer downloadStream.Close(true)

	c.Header("Content-Length", strconv.FormatInt(downloadStream.Size(), 10))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.QueryEscape(downloadStream.Name())))
	c.Status(http.StatusOK)

	b := make([]byte, constants.IOBufferSize)
loop:
	for {
		select {
		case <-c.Request.Context().Done():
			return
		default:
			readedBytes, err := downloadStream.Read(b)
			c.Writer.Write(b[:readedBytes])
			if err == io.EOF {
				break loop
			}
		}
	}
}

func (w *WebServer) getWorkers(c *gin.Context) {
	workers, err := w.scheduler.GetWorkers(w.ctx)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, workers)
}

func (w *WebServer) checksum(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("job ID parameter not found"), 404)
		return
	}

	checksum, err := w.scheduler.GetChecksum(c.Request.Context(), id)
	if webError(c, err, 404) {
		return
	}
	c.Header("Content-Length", strconv.Itoa(len(checksum)))
	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, checksum)
}

type WebServerConfig struct {
	Port          int                  `mapstructure:"port"`
	Token         string               `mapstructure:"token"`
	WebhookConfig *model.WebhookConfig `mapstructure:"webhook"`
	AuthConfig    *auth.AuthConfig     `mapstructure:"auth"`
}

func NewWebServer(config WebServerConfig, scheduler scheduler.Scheduler, w *watcher.Watcher, scanner *scanner.Scanner, repo repository.Repository, authService *auth.AuthService) *WebServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	webServer := &WebServer{
		WebServerConfig: config,
		scheduler:       scheduler,
		scanner:         scanner,
		router:          r,
		webhookConfig:   config.WebhookConfig,
		repo:            repo,
		authService:     authService,
	}

	if w != nil {
		webServer.watcherHandler = watcher.NewHandler(w)
	}

	if config.WebhookConfig != nil && config.WebhookConfig.Enabled {
		registry := webhook.NewDefaultHandlerRegistry()
		webServer.webhookHandler = webhook.NewHTTPHandlerWithQueuer(registry, repo, scheduler)
	}

	r.GET("/-/healthy", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	r.HEAD("/-/healthy", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	authGroup := r.Group("/auth")
	authGroup.GET("/login", webServer.authLogin)
	authGroup.GET("/callback", webServer.authCallback)
	authGroup.GET("/logout", webServer.authLogout)

	api := r.Group("/api/v1")
	api.Use(webServer.authMiddleware())
	api.GET("/job/", webServer.getJobs)
	api.POST("/job/", webServer.addJob)
	api.GET("/job/:id", webServer.getJobByID)
	api.DELETE("/job/:id", webServer.deleteJob)
	api.PATCH("/job/:id/priority", webServer.updateJobPriority)
	api.GET("/job/:id/download", webServer.authWorkerMiddleware(), webServer.download)
	api.GET("/job/:id/checksum", webServer.authWorkerMiddleware(), webServer.checksum)
	api.POST("/job/:id/upload", webServer.authWorkerMiddleware(), webServer.upload)

	api.GET("/workers/", webServer.getWorkers)

	api.GET("/watcher/status", webServer.getWatcherStatus)
	api.GET("/watcher/detections", webServer.getWatcherDetections)
	api.POST("/watcher/paths", webServer.addWatcherPath)
	api.DELETE("/watcher/paths", webServer.removeWatcherPath)
	api.GET("/watcher/enabled", webServer.getWatcherEnabled)

	apiTokens := api.Group("/tokens")
	apiTokens.GET("/", webServer.listAPITokens)
	apiTokens.POST("/", webServer.createAPIToken)
	apiTokens.DELETE("/:id", webServer.deleteAPIToken)

	webhookGroup := r.Group("/api/v1/webhook")
	webhookGroup.POST("/radarr", webServer.webhookAuthMiddleware(string(model.WebhookProviderRadarr)), webServer.handleWebhook)
	webhookGroup.POST("/sonarr", webServer.webhookAuthMiddleware(string(model.WebhookProviderSonarr)), webServer.handleWebhook)
	webhookGroup.POST("/test", webServer.handleWebhookTest)
	webhookGroup.GET("/events", webServer.authMiddleware(), webServer.getWebhookEvents)
	webhookGroup.GET("/events/:id", webServer.authMiddleware(), webServer.getWebhookEventByID)

	r.GET("/ws/job", webServer.AuthParamFunc(webServer.getJobsUpdates))

	if scanner != nil {
		api.GET("/scanner/status", webServer.getScannerStatus)
		api.POST("/scanner/scan", webServer.triggerScan)
		api.GET("/scanner/history", webServer.getScanHistory)
		r.GET("/ws/scanner", webServer.AuthParamFunc(webServer.getScannerUpdates))
	}

	ui.AddRoutes(r)

	return webServer
}

func (w *WebServer) Run(wg *sync.WaitGroup, ctx context.Context) {
	w.ctx = ctx
	helper.Info("starting webserver")
	w.start()
	helper.Info("started webserver")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		helper.Info("stopping webserver")
		w.stop(ctx)
		wg.Done()
	}()
}

func (w *WebServer) start() {
	go func() {
		err := w.router.Run(":" + strconv.Itoa(w.Port))
		if err != nil {
			helper.Panic(err)
		}
	}()
}

func (w *WebServer) stop(ctx context.Context) {
	server := &http.Server{Addr: ":" + strconv.Itoa(w.Port), Handler: w.router}
	if err := server.Shutdown(ctx); err != nil {
		helper.Panic(err)
	}
}

func (w *WebServer) AuthHeaderFunc(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Missing Authorization header"})
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid Authorization header format"})
			return
		}

		t := strings.TrimPrefix(authHeader, bearerPrefix)

		if t != w.Token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid token"})
			return
		}

		handler(c)
	}
}

func (w *WebServer) AuthParamFunc(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")

		if token == "" || token != w.Token {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		handler(c)
	}
}

func webError(c *gin.Context, err error, code int) bool {
	if err != nil {
		c.AbortWithStatusJSON(code, gin.H{"error": err.Error()})
		return true
	}
	return false
}

func (w *WebServer) getWatcherStatus(c *gin.Context) {
	if w.watcherHandler == nil {
		c.JSON(http.StatusOK, gin.H{"active": false, "watched_paths": []string{}, "files_detected": 0, "files_queued": 0})
		return
	}
	w.watcherHandler.GetStatus(c)
}

func (w *WebServer) getWatcherDetections(c *gin.Context) {
	if w.watcherHandler == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	w.watcherHandler.GetDetections(c)
}

func (w *WebServer) addWatcherPath(c *gin.Context) {
	if w.watcherHandler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "watcher not initialized"})
		return
	}
	w.watcherHandler.AddPath(c)
}

func (w *WebServer) removeWatcherPath(c *gin.Context) {
	if w.watcherHandler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "watcher not initialized"})
		return
	}
	w.watcherHandler.RemovePath(c)
}

func (w *WebServer) getWatcherEnabled(c *gin.Context) {
	if w.watcherHandler == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}
	w.watcherHandler.IsEnabled(c)
}

func (w *WebServer) getScannerStatus(c *gin.Context) {
	if w.scanner == nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false, "is_scanning": false})
		return
	}
	status := w.scanner.GetStatus()
	c.JSON(http.StatusOK, status)
}

func (w *WebServer) triggerScan(c *gin.Context) {
	if w.scanner == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scanner not configured"})
		return
	}
	w.scanner.TriggerScan()
	c.JSON(http.StatusAccepted, gin.H{"message": "scan triggered"})
}

func (w *WebServer) getScanHistory(c *gin.Context) {
	if w.scanner == nil {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	history, err := w.scanner.GetHistory(w.ctx, 10)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, history)
}

func (w *WebServer) getScannerUpdates(c *gin.Context) {
	if w.scanner == nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	conn, err := w.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		helper.Errorf("failed to upgrade: %s", err)
		return
	}
	defer conn.Close()
	helper.Debug("scanner websocket connected")

	ch := w.scanner.GetNotificationChan()
	for {
		select {
		case <-w.ctx.Done():
			return
		case notification, ok := <-ch:
			if !ok {
				return
			}
			jsonBytes, err := json.Marshal(notification)
			if err != nil {
				helper.Errorf("scanner notification cannot be marshaled: %s", err)
				return
			}
			helper.Debugf("sending scanner update: %+v", notification)
			conn.WriteMessage(websocket.TextMessage, jsonBytes)
		}
	}
}

func (w *WebServer) handleWebhook(c *gin.Context) {
	if w.webhookHandler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhooks not configured"})
		return
	}
	w.webhookHandler.HandleWebhook(c)
}

func (w *WebServer) handleWebhookTest(c *gin.Context) {
	if w.webhookHandler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhooks not configured"})
		return
	}
	w.webhookHandler.HandleTest(c)
}

func (w *WebServer) webhookAuthMiddleware(providerName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if w.webhookConfig == nil || !w.webhookConfig.Enabled {
			c.Next()
			return
		}

		apiKey := c.GetHeader("X-Api-Key")
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			return
		}

		if !w.webhookConfig.ValidateAuth(providerName, apiKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		c.Next()
	}
}

func (w *WebServer) getWebhookEvents(c *gin.Context) {
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	source := c.Query("source")
	eventType := c.Query("event_type")
	status := c.Query("status")

	events, err := w.scheduler.GetWebhookEvents(c.Request.Context(), limit, source, eventType, status)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, events)
}

func (w *WebServer) getWebhookEventByID(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		webError(c, fmt.Errorf("event ID parameter not found"), http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		webError(c, fmt.Errorf("invalid event ID"), http.StatusBadRequest)
		return
	}

	event, err := w.scheduler.GetWebhookEvent(c.Request.Context(), id)
	if err != nil {
		webError(c, err, http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, event)
}

func (w *WebServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if w.authService == nil {
			if w.Token != "" {
				authHeader := c.GetHeader("Authorization")
				const bearerPrefix = "Bearer "
				if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
					return
				}
				token := strings.TrimPrefix(authHeader, bearerPrefix)
				if token != w.Token {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
					return
				}
				c.Set("auth_scope", model.ScopeAdmin)
				c.Next()
				return
			}
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		apiToken, session, err := w.authService.ValidateBearerToken(c.Request.Context(), authHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		if apiToken != nil {
			c.Set("auth_token_id", apiToken.ID)
			c.Set("auth_scope", apiToken.Scope)
		} else if session != nil {
			c.Set("auth_user_id", session.UserID)
			c.Set("auth_user_email", session.Email)
			c.Set("auth_scope", session.Scope)
		}

		c.Next()
	}
}

func (w *WebServer) authWorkerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if w.authService == nil {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		apiToken, session, err := w.authService.ValidateBearerToken(c.Request.Context(), authHeader)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var scope model.TokenScope
		if apiToken != nil {
			scope = apiToken.Scope
		} else if session != nil {
			scope = session.Scope
		}

		if scope != model.ScopeWorker && scope != model.ScopeAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: worker or admin scope required"})
			return
		}

		c.Next()
	}
}

func (w *WebServer) authLogin(c *gin.Context) {
	if w.authService == nil || !w.authService.IsOIDCEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC not configured"})
		return
	}

	oauth2Config := w.authService.GetOAuth2Config()
	state, err := auth.GenerateToken()
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.SetCookie("auth_state", state, 300, "/", "", false, true)
	url := oauth2Config.AuthCodeURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (w *WebServer) authCallback(c *gin.Context) {
	if w.authService == nil || !w.authService.IsOIDCEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC not configured"})
		return
	}

	state := c.Query("state")
	cookieState, err := c.Cookie("auth_state")
	if err != nil || state != cookieState {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid state"})
		return
	}
	c.SetCookie("auth_state", "", -1, "/", "", false, true)

	code := c.Query("code")
	token, err := w.authService.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	userInfo, err := w.authService.GetUserInfo(c.Request.Context(), oauth2.StaticTokenSource(token))
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	sessionToken, err := w.authService.CreateSession(userInfo.Subject, userInfo.Email, userInfo.Name)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.SetCookie("gearr_session", sessionToken, 86400, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (w *WebServer) authLogout(c *gin.Context) {
	c.SetCookie("gearr_session", "", -1, "/", "", false, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (w *WebServer) listAPITokens(c *gin.Context) {
	if w.authService == nil || !w.authService.IsAPITokensEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "API tokens not configured"})
		return
	}

	tokens, err := w.authService.ListAPITokens(c.Request.Context())
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, tokens)
}

func (w *WebServer) createAPIToken(c *gin.Context) {
	if w.authService == nil || !w.authService.IsAPITokensEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "API tokens not configured"})
		return
	}

	var req struct {
		Name      string           `json:"name"`
		Scope     model.TokenScope `json:"scope"`
		ExpiresAt *string          `json:"expires_at,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if !model.IsValidScope(req.Scope) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid scope"})
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid expires_at format"})
			return
		}
		expiresAt = &t
	}

	createdBy := ""
	if userID, exists := c.Get("auth_user_id"); exists {
		createdBy = userID.(string)
	} else if tokenID, exists := c.Get("auth_token_id"); exists {
		createdBy = tokenID.(string)
	}

	token, rawToken, err := w.authService.CreateAPIToken(c.Request.Context(), req.Name, req.Scope, createdBy, expiresAt)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         token.ID,
		"name":       token.Name,
		"scope":      token.Scope,
		"token":      rawToken,
		"created_at": token.CreatedAt,
		"expires_at": token.ExpiresAt,
	})
}

func (w *WebServer) deleteAPIToken(c *gin.Context) {
	if w.authService == nil || !w.authService.IsAPITokensEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "API tokens not configured"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "token ID required"})
		return
	}

	err := w.authService.DeleteAPIToken(c.Request.Context(), id)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

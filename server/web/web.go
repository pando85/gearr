package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gearr/helper"
	"gearr/internal/constants"
	"gearr/model"
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

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

	if req.Priority < 0 || req.Priority > 3 {
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
}

func NewWebServer(config WebServerConfig, scheduler scheduler.Scheduler, w *watcher.Watcher, scanner *scanner.Scanner, repo repository.Repository) *WebServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	webServer := &WebServer{
		WebServerConfig: config,
		scheduler:       scheduler,
		scanner:         scanner,
		router:          r,
		webhookConfig:   config.WebhookConfig,
		repo:            repo,
	}

	if w != nil {
		webServer.watcherHandler = watcher.NewHandler(w)
	}

	if config.WebhookConfig != nil && config.WebhookConfig.Enabled {
		registry := webhook.NewDefaultHandlerRegistry()
		if repo != nil {
			webServer.webhookHandler = webhook.NewHTTPHandlerWithLogger(registry, repo)
		} else {
			webServer.webhookHandler = webhook.NewHTTPHandler(registry)
		}
	}

	r.GET("/-/healthy", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	r.HEAD("/-/healthy", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	api := r.Group("/api/v1")
	api.GET("/job/", webServer.AuthHeaderFunc(webServer.getJobs))
	api.POST("/job/", webServer.AuthHeaderFunc(webServer.addJob))
	api.GET("/job/:id", webServer.AuthHeaderFunc(webServer.getJobByID))
	api.DELETE("/job/:id", webServer.AuthHeaderFunc(webServer.deleteJob))
	api.PATCH("/job/:id/priority", webServer.AuthHeaderFunc(webServer.updateJobPriority))
	api.GET("/job/:id/download", webServer.download)
	api.GET("/job/:id/checksum", webServer.checksum)
	api.POST("/job/:id/upload", webServer.upload)

	api.GET("/workers/", webServer.AuthHeaderFunc(webServer.getWorkers))

	api.GET("/watcher/status", webServer.AuthHeaderFunc(webServer.getWatcherStatus))
	api.GET("/watcher/detections", webServer.AuthHeaderFunc(webServer.getWatcherDetections))
	api.POST("/watcher/paths", webServer.AuthHeaderFunc(webServer.addWatcherPath))
	api.DELETE("/watcher/paths", webServer.AuthHeaderFunc(webServer.removeWatcherPath))
	api.GET("/watcher/enabled", webServer.AuthHeaderFunc(webServer.getWatcherEnabled))

	webhook := r.Group("/api/v1/webhook")
	webhook.POST("/radarr", webServer.webhookAuthMiddleware(string(model.WebhookProviderRadarr)), webServer.handleWebhook)
	webhook.POST("/sonarr", webServer.webhookAuthMiddleware(string(model.WebhookProviderSonarr)), webServer.handleWebhook)
	webhook.POST("/test", webServer.handleWebhookTest)
	webhook.GET("/events", webServer.AuthHeaderFunc(webServer.getWebhookEvents))
	webhook.GET("/events/:id", webServer.AuthHeaderFunc(webServer.getWebhookEventByID))

	r.GET("/ws/job", webServer.AuthParamFunc(webServer.getJobsUpdates))

	if scanner != nil {
		api.GET("/scanner/status", webServer.AuthHeaderFunc(webServer.getScannerStatus))
		api.POST("/scanner/scan", webServer.AuthHeaderFunc(webServer.triggerScan))
		api.GET("/scanner/history", webServer.AuthHeaderFunc(webServer.getScanHistory))
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

package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gearr/helper"
	"gearr/internal/constants"
	"gearr/model"
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
	scheduler       scheduler.Scheduler
	scanner         *scanner.Scanner
	router          *gin.Engine
	ctx             context.Context
	upgrader        websocket.Upgrader
	watcherHandler  *watcher.Handler
	webhookConfig   model.WebhookConfig
	webhookRegistry *webhook.HandlerRegistry
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
	Port  int    `mapstructure:"port"`
	Token string `mapstructure:"token"`
}

func NewWebServer(config WebServerConfig, scheduler scheduler.Scheduler, w *watcher.Watcher, scanner *scanner.Scanner, webhookConfig model.WebhookConfig) *WebServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	registry := webhook.NewHandlerRegistry()
	registry.Register(webhook.NewRadarrHandler())
	registry.Register(webhook.NewSonarrHandler())

	webServer := &WebServer{
		WebServerConfig: config,
		scheduler:       scheduler,
		scanner:         scanner,
		router:          r,
		webhookConfig:   webhookConfig,
		webhookRegistry: registry,
	}

	if w != nil {
		webServer.watcherHandler = watcher.NewHandler(w)
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
	api.GET("/job/:id/download", webServer.download)
	api.GET("/job/:id/checksum", webServer.checksum)
	api.POST("/job/:id/upload", webServer.upload)

	api.GET("/workers/", webServer.AuthHeaderFunc(webServer.getWorkers))

	api.GET("/watcher/status", webServer.AuthHeaderFunc(webServer.getWatcherStatus))
	api.GET("/watcher/detections", webServer.AuthHeaderFunc(webServer.getWatcherDetections))
	api.POST("/watcher/paths", webServer.AuthHeaderFunc(webServer.addWatcherPath))
	api.DELETE("/watcher/paths", webServer.AuthHeaderFunc(webServer.removeWatcherPath))
	api.GET("/watcher/enabled", webServer.AuthHeaderFunc(webServer.getWatcherEnabled))

	r.GET("/ws/job", webServer.AuthParamFunc(webServer.getJobsUpdates))

	if scanner != nil {
		api.GET("/scanner/status", webServer.AuthHeaderFunc(webServer.getScannerStatus))
		api.POST("/scanner/scan", webServer.AuthHeaderFunc(webServer.triggerScan))
		api.GET("/scanner/history", webServer.AuthHeaderFunc(webServer.getScanHistory))
		r.GET("/ws/scanner", webServer.AuthParamFunc(webServer.getScannerUpdates))
	}

	if webhookConfig.Enabled {
		api.POST("/webhook/radarr", webServer.handleRadarrWebhook)
		api.POST("/webhook/sonarr", webServer.handleSonarrWebhook)
		api.POST("/webhook/test", webServer.handleTestWebhook)
	}

	api.PATCH("/job/:id/priority", webServer.AuthHeaderFunc(webServer.updateJobPriority))

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

func (w *WebServer) handleRadarrWebhook(c *gin.Context) {
	if !w.webhookConfig.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhooks not enabled"})
		return
	}

	apiKey := c.GetHeader("X-Api-Key")
	if apiKey == "" {
		apiKey = c.Query("apikey")
	}

	radarrProvider := w.webhookConfig.GetProvider("radarr")
	if radarrProvider == nil || !radarrProvider.ValidateAPIKey(apiKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	handler := w.webhookRegistry.GetHandler(webhook.SourceRadarr, webhook.EventDownload)
	if handler == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no handler found for Radarr webhook"})
		return
	}

	payload, err := handler.Parse(c.Request.Context(), body)
	if err != nil {
		helper.Errorf("failed to parse Radarr webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse webhook payload"})
		return
	}

	result, err := handler.Process(c.Request.Context(), payload)
	if err != nil {
		helper.Errorf("failed to process Radarr webhook: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

	if result.Accepted && len(result.Files) > 0 {
		w.processWebhookFiles(result.Files)
	}

	c.JSON(http.StatusOK, gin.H{
		"accepted":    result.Accepted,
		"files":       result.Files,
		"skip_reason": result.SkipReason,
		"message":     "webhook processed",
	})
}

func (w *WebServer) handleSonarrWebhook(c *gin.Context) {
	if !w.webhookConfig.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhooks not enabled"})
		return
	}

	apiKey := c.GetHeader("X-Api-Key")
	if apiKey == "" {
		apiKey = c.Query("apikey")
	}

	sonarrProvider := w.webhookConfig.GetProvider("sonarr")
	if sonarrProvider == nil || !sonarrProvider.ValidateAPIKey(apiKey) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	handler := w.webhookRegistry.GetHandler(webhook.SourceSonarr, webhook.EventDownload)
	if handler == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no handler found for Sonarr webhook"})
		return
	}

	payload, err := handler.Parse(c.Request.Context(), body)
	if err != nil {
		helper.Errorf("failed to parse Sonarr webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse webhook payload"})
		return
	}

	result, err := handler.Process(c.Request.Context(), payload)
	if err != nil {
		helper.Errorf("failed to process Sonarr webhook: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

	if result.Accepted && len(result.Files) > 0 {
		w.processWebhookFiles(result.Files)
	}

	c.JSON(http.StatusOK, gin.H{
		"accepted":    result.Accepted,
		"files":       result.Files,
		"skip_reason": result.SkipReason,
		"message":     "webhook processed",
	})
}

func (w *WebServer) handleTestWebhook(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	handler := w.webhookRegistry.GetHandler(webhook.SourceRadarr, webhook.EventTest)
	if handler == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no test handler found"})
		return
	}

	payload, err := handler.Parse(c.Request.Context(), body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse test webhook payload"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accepted":    true,
		"source_type": payload.SourceType,
		"event_type":  payload.EventType,
		"message":     "test webhook received successfully",
	})
}

func (w *WebServer) processWebhookFiles(files []webhook.File) {
	for _, file := range files {
		if file.Path == "" {
			continue
		}

		jobRequest := &model.JobRequest{
			SourcePath:      file.Path,
			DestinationPath: "",
			Priority:        0,
		}

		_, err := w.scheduler.ScheduleJobRequest(w.ctx, jobRequest)
		if err != nil {
			if errors.Is(err, model.ErrJobExists) {
				helper.Debugf("job already exists for file: %s", file.Path)
				continue
			}
			helper.Errorf("failed to schedule job from webhook for file %s: %v", file.Path, err)
			continue
		}

		helper.Infof("queued job from webhook for file: %s", file.Path)
	}
}

func (w *WebServer) updateJobPriority(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("job ID parameter not found"), http.StatusBadRequest)
		return
	}

	var request struct {
		Priority int `json:"priority" binding:"required,min=0,max=3"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		webError(c, err, http.StatusBadRequest)
		return
	}

	err := w.scheduler.UpdateJobPriority(w.ctx, id, request.Priority)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       id,
		"priority": request.Priority,
		"message":  "job priority updated successfully",
	})
}

package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"transcoder/model"
	"transcoder/server/scheduler"
	"transcoder/server/web/ui"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type WebServer struct {
	WebServerConfig
	scheduler scheduler.Scheduler
	router    *gin.Engine
	ctx       context.Context
}

func (w *WebServer) addJobs(c *gin.Context) {
	var jobRequest model.JobRequest
	if err := c.ShouldBindJSON(&jobRequest); err != nil {
		webError(c, err, 500)
		return
	}

	scheduleJobResults, err := w.scheduler.ScheduleJobRequests(w.ctx, &jobRequest)
	if webError(c, err, 500) {
		return
	}

	c.JSON(http.StatusOK, scheduleJobResults)
}

func getPageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	return page, pageSize
}

func (w *WebServer) getJobs(c *gin.Context) {
	page, pageSize := getPageParams(c)

	videos, err := w.scheduler.GetJobs(w.ctx, page, pageSize)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, videos)
}

func (w *WebServer) getJobByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("Job ID parameter not found"), 404)
		return
	}

	video, err := w.scheduler.GetJob(w.ctx, id)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, video)
}

func (w *WebServer) deleteJob(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("Job ID parameter not found"), 404)
		return
	}

	err := w.scheduler.DeleteJob(w.ctx, id)
	if err != nil {
		webError(c, err, http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (w *WebServer) upload(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		webError(c, fmt.Errorf("Job ID parameter not found"), 404)
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

	size, err := strconv.ParseUint(c.GetHeader("Content-Length"), 10, 64)
	checksum := c.GetHeader("checksum")
	if checksum == "" {
		webError(c, fmt.Errorf("checksum is mandatory in the headers"), 403)
		return
	}

	b := make([]byte, 131072)
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
		webError(c, fmt.Errorf("Job ID parameter not found"), 404)
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

	b := make([]byte, 131072)
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
		webError(c, fmt.Errorf("Job ID parameter not found"), 404)
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

func NewWebServer(config WebServerConfig, scheduler scheduler.Scheduler) *WebServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	webServer := &WebServer{
		WebServerConfig: config,
		scheduler:       scheduler,
		router:          r,
	}

	r.GET("/-/healthy", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	r.HEAD("/-/healthy", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	api := r.Group("/api/v1")
	api.GET("/job/", webServer.AuthFunc(webServer.getJobs))
	api.POST("/job/", webServer.AuthFunc(webServer.addJobs))
	api.GET("/job/:id", webServer.AuthFunc(webServer.getJobByID))
	api.DELETE("/job/:id", webServer.AuthFunc(webServer.deleteJob))
	api.GET("/job/:id/download", webServer.download)
	api.GET("/job/:id/checksum", webServer.checksum)
	api.POST("/job/:id/upload", webServer.upload)

	api.GET("/workers/", webServer.AuthFunc(webServer.getWorkers))

	ui.AddRoutes(r)

	return webServer
}

func (w *WebServer) Run(wg *sync.WaitGroup, ctx context.Context) {
	w.ctx = ctx
	log.Info("starting webserver")
	w.start()
	log.Info("started webserver")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("stopping webserver")
		w.stop(ctx)
		wg.Done()
	}()
}

func (w *WebServer) start() {
	go func() {
		err := w.router.Run(":" + strconv.Itoa(w.Port))
		if err != nil {
			log.Panic(err)
		}
	}()
}

func (w *WebServer) stop(ctx context.Context) {
	server := &http.Server{Addr: ":" + strconv.Itoa(w.Port), Handler: w.router}
	if err := server.Shutdown(ctx); err != nil {
		log.Panic(err)
	}
}

func (w *WebServer) AuthFunc(handler gin.HandlerFunc) gin.HandlerFunc {
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

func webError(c *gin.Context, err error, code int) bool {
	if err != nil {
		c.AbortWithStatusJSON(code, gin.H{"error": err.Error()})
		return true
	}
	return false
}

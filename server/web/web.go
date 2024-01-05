package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"transcoder/model"
	"transcoder/server/scheduler"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type WebServer struct {
	WebServerConfig
	scheduler scheduler.Scheduler
	srv       http.Server
	ctx       context.Context
}

func (W *WebServer) cancelJob(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	values := request.URL.Query()
	uuid := values.Get("uuid")
	if uuid == "" {
		webError(writer, fmt.Errorf("UUID get parameter not found"), 404)
	}
	var err error
	err = W.scheduler.CancelJob(request.Context(), uuid)

	if err != nil {
		if errors.Is(err, scheduler.ErrorJobNotFound) {
			webError(writer, err, 404)
		} else {
			webError(writer, err, 500)
		}
		return
	}
}

func (W *WebServer) addJobs(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	jobRequest := &model.JobRequest{}
	err := json.NewDecoder(request.Body).Decode(jobRequest)
	if err != nil {
		webError(writer, err, 500)
		return
	}

	scheduleJobResults, err := W.scheduler.ScheduleJobRequests(W.ctx, jobRequest)
	if webError(writer, err, 500) {
		return
	}

	if webError(writer, err, 500) {
		return
	}
	b, err := json.MarshalIndent(scheduleJobResults, "", "\t")
	if err != nil {
		if webError(writer, err, 500) {
			return
		}
	}
	writer.WriteHeader(200)
	writer.Write(b)
}

func (W *WebServer) upload(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	uuid := values.Get("uuid")
	if uuid == "" {
		webError(writer, fmt.Errorf("UUID get parameter not found"), 404)
	}
	uploadStream, err := W.scheduler.GetUploadJobWriter(request.Context(), uuid)
	if errors.Is(err, scheduler.ErrorStreamNotAllowed) {
		webError(writer, err, 403)
		return
	} else if errors.Is(err, scheduler.ErrorJobNotFound) {
		webError(writer, err, 404)
		return
	} else if webError(writer, err, 500) {
		return
	}
	defer uploadStream.Close(false)

	size, err := strconv.ParseUint(request.Header.Get("Content-Length"), 10, 64)
	checksum := request.Header.Get("checksum")
	if checksum == "" {
		webError(writer, fmt.Errorf("checksum is mandatory in the headers"), 403)
		return
	}

	b := make([]byte, 131072)
	reader := request.Body
	var readed uint64
loop:
	for {
		select {
		case <-request.Context().Done():
			return
		default:
			readedBytes, err := reader.Read(b)
			readed += uint64(readedBytes)
			uploadStream.Write(b[0:readedBytes])
			//TODO check error here?
			if err == io.EOF {
				break loop
			}
		}
	}
	if size != readed {
		defer uploadStream.Clean()
		webError(writer, fmt.Errorf("invalid size, expected %d, received %d", size, readed), 400)
		return
	}
	checksumUpload := uploadStream.GetHash()
	if checksumUpload != checksum {
		defer uploadStream.Clean()
		webError(writer, fmt.Errorf("invalid checksum, received %s, calculated %s", checksum, checksumUpload), 400)
		return
	}
	writer.WriteHeader(201)
}

func (W *WebServer) download(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	uuid := values.Get("uuid")
	if uuid == "" {
		webError(writer, fmt.Errorf("UUID get parameter not found"), 404)
	}
	downloadStream, err := W.scheduler.GetDownloadJobWriter(request.Context(), uuid)
	if errors.Is(err, scheduler.ErrorStreamNotAllowed) {
		webError(writer, err, 403)
		return
	} else if errors.Is(err, scheduler.ErrorJobNotFound) {
		webError(writer, err, 404)
		return
	} else if webError(writer, err, 500) {
		return
	}
	defer downloadStream.Close(true)

	writer.Header().Set("Content-Length", strconv.FormatInt(downloadStream.Size(), 10))
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.QueryEscape(downloadStream.Name())))
	writer.WriteHeader(200)
	b := make([]byte, 131072)
loop:
	for {
		select {
		case <-request.Context().Done():
			return
		default:
			readedBytes, err := downloadStream.Read(b)
			writer.Write(b[0:readedBytes])
			if err == io.EOF {
				break loop
			}
		}
	}
}

func (W *WebServer) checksum(writer http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	uuid := values.Get("uuid")
	if uuid == "" {
		webError(writer, fmt.Errorf("UUID get parameter not found"), 404)
		return
	}
	checksum, err := W.scheduler.GetChecksum(request.Context(), uuid)
	if webError(writer, err, 404) {
		return
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(checksum)))
	writer.Header().Set("Content-Type", "text/plain")
	writer.WriteHeader(200)
	writer.Write([]byte(checksum))
}

type WebServerConfig struct {
	Port  int    `mapstructure:"port"`
	Token string `mapstructure:"token"`
}

func NewWebServer(config WebServerConfig, scheduler scheduler.Scheduler) *WebServer {
	rtr := mux.NewRouter()
	webServer := &WebServer{
		WebServerConfig: config,
		scheduler:       scheduler,
		srv: http.Server{
			Addr:    ":" + strconv.Itoa(config.Port),
			Handler: rtr,
		},
	}
	rtr.Handle("/api/v1/job/", webServer.AuthFunc(webServer.addJobs)).Methods("POST")
	rtr.HandleFunc("/api/v1/job/cancel", webServer.cancelJob).Methods("GET")
	rtr.HandleFunc("/api/v1/download", webServer.download).Methods("GET")
	rtr.HandleFunc("/api/v1/checksum", webServer.checksum).Methods("GET")
	rtr.HandleFunc("/api/v1/upload", webServer.upload).Methods("POST", "PUT")
	return webServer
}

func (W *WebServer) Run(wg *sync.WaitGroup, ctx context.Context) {
	W.ctx = ctx
	log.Info("Starting WebServer...")
	W.start()
	log.Info("Started WebServer...")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("Stopping WebServer...")
		W.stop(ctx)
		wg.Done()
	}()
}

func (W *WebServer) start() {
	go func() {
		err := W.srv.ListenAndServe()
		if err != nil {
			log.Panic(err)
		}
	}()
}

func (W *WebServer) stop(ctx context.Context) {
	if err := W.srv.Shutdown(ctx); err != nil {
		log.Panic(err)
	}
}

func (S *WebServer) AuthFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := r.URL.Query().Get("token")

		if t != S.Token {
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}
		handler(w, r)
	}
}

func webError(writer http.ResponseWriter, err error, code int) bool {
	if err != nil {
		writer.WriteHeader(code)
		writer.Write([]byte(err.Error()))
		return true
	}
	return false
}

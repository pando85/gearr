package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"transcoder/helper"
	"transcoder/model"
	"transcoder/server/queue"
	"transcoder/server/repository"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/vansante/go-ffprobe.v2"
)

var (
	x264ex = regexp.MustCompile(`(?i)(((x|h)264)|mpeg-4|mpeg-1|mpeg-2|mpeg|xvid|divx|vc-1|av1|vp8|vp9|wmv3|mp43)`)
	ac3ex  = regexp.MustCompile(`(?i)(ac3|eac3|pcm|flac|mp2|dts|mp2|mp3|truehd|wma|vorbis|opus|mpeg audio)`)
)

type Scheduler interface {
	Run(wg *sync.WaitGroup, ctx context.Context)
	ScheduleJobRequests(ctx context.Context, jobRequest *model.JobRequest) (*ScheduleJobRequestResult, error)
	GetUploadJobWriter(ctx context.Context, uuid string) (*UploadJobStream, error)
	GetDownloadJobWriter(ctx context.Context, uuid string) (*DownloadJobStream, error)
	GetChecksum(ctx context.Context, uuid string) (string, error)
	CancelJob(ctx context.Context, uuid string) error
}

type SchedulerConfig struct {
	ScheduleTime time.Duration `mapstructure:"scheduleTime"`
	JobTimeout   time.Duration `mapstructure:"jobTimeout"`
	DownloadPath string        `mapstructure:"downloadPath"`
	UploadPath   string        `mapstructure:"uploadPath"`
	Domain       *url.URL
	MinFileSize  int64 `mapstructure:"minFileSize"`
	checksums    map[string][]byte
}

type RuntimeScheduler struct {
	config          SchedulerConfig
	repo            repository.Repository
	queue           queue.BrokerServer
	checksumChan    chan PathChecksum
	pathChecksumMap map[string]string
}

func NewScheduler(config SchedulerConfig, repo repository.Repository, queue queue.BrokerServer) (*RuntimeScheduler, error) {
	runtimeScheduler := &RuntimeScheduler{
		config:          config,
		repo:            repo,
		queue:           queue,
		checksumChan:    make(chan PathChecksum),
		pathChecksumMap: make(map[string]string),
	}

	return runtimeScheduler, nil
}

func (R *RuntimeScheduler) Run(wg *sync.WaitGroup, ctx context.Context) {
	log.Info("starting scheduler")
	R.start(ctx)
	log.Info("started scheduler")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("stopping scheduler")
		R.stop()
		wg.Done()
	}()
}

func (R *RuntimeScheduler) start(ctx context.Context) {
	go R.schedule(ctx)
}

func (R *RuntimeScheduler) schedule(ctx context.Context) {
	jobEventConsumerChan := R.queue.ReceiveJobEvent()
	for {
		select {
		case <-ctx.Done():
			return
		case jobEvent, ok := <-jobEventConsumerChan:
			if !ok {
				return
			}
			if jobEvent.EventType == model.NotificationEvent && jobEvent.NotificationType == model.JobNotification && jobEvent.Status == model.CompletedNotificationStatus {
				video, err := R.repo.GetJob(ctx, jobEvent.Id.String())
				if err != nil {
					log.Error(err)
					continue
				}
				sourcePath := filepath.Join(R.config.DownloadPath, video.SourcePath)
				target := filepath.Join(R.config.DownloadPath, video.DestinationPath)
				if _, err := os.Stat(target); err != nil {
					log.Warnf("job %s completed, source file %s can not be removed because target file does not exists", jobEvent.Id.String(), sourcePath)
					continue
				}
				log.Infof("job %s completed, removing source file %s", jobEvent.Id.String(), sourcePath)
				err = os.Remove(sourcePath)
				if err != nil {
					log.Error(err)
				}

			}
		case checksumPath := <-R.checksumChan:
			R.pathChecksumMap[checksumPath.path] = checksumPath.checksum
		case <-time.After(R.config.ScheduleTime):
			taskEvents, err := R.repo.GetTimeoutJobs(ctx, R.config.JobTimeout)
			if err != nil {
				log.Error(err)
			}
			for _, taskEvent := range taskEvents {
				if taskEvent.Status == model.StartedNotificationStatus {
					log.Infof("rescheduling %s after job timeout", taskEvent.Id.String())
					video, err := R.repo.GetJob(ctx, taskEvent.Id.String())
					if err != nil {
						log.Error(err)
						continue
					}
					jobRequest := &model.JobRequest{
						SourcePath:      video.SourcePath,
						DestinationPath: video.DestinationPath,
						ForceExecuting:  true,
						Priority:        9,
					}
					video, err = R.scheduleJobRequest(ctx, jobRequest)
					if err != nil {
						log.Error(err)
					}
				}
			}
		}
	}
}

type JobRequestResult struct {
	jobRequest *model.JobRequest
	errors     []string
}

func (R *RuntimeScheduler) createNewJobRequestByJobRequestDirectory(ctx context.Context, parentJobRequest *model.JobRequest, searchJobRequestChan chan<- *JobRequestResult) {
	defer close(searchJobRequestChan)
	filepath.Walk(filepath.Join(R.config.DownloadPath, parentJobRequest.SourcePath), func(pathFile string, f os.FileInfo, err error) error {
		var jobRequestErrors []string
		select {
		case <-ctx.Done():
			return fmt.Errorf("search for new Jobs canceled")
		default:
			if f.IsDir() {
				return nil
			}
			if f.Size() < R.config.MinFileSize {
				jobRequestErrors = append(jobRequestErrors, fmt.Sprintf("%s File size must be bigger than %d", pathFile, R.config.MinFileSize))
			}
			extension := filepath.Ext(f.Name())[1:]
			if !helper.ValidExtension(extension) {
				jobRequestErrors = append(jobRequestErrors, fmt.Sprintf("%s Invalid Extension %s", pathFile, extension))
			}

			relativePathSource, err := filepath.Rel(R.config.DownloadPath, filepath.FromSlash(pathFile))
			if err != nil {
				jobRequestErrors = append(jobRequestErrors, err.Error())
			}

			relativePathTarget := formatTargetName(relativePathSource)
			if relativePathTarget == relativePathSource {
				ext := filepath.Ext(relativePathTarget)
				relativePathTarget = strings.Replace(relativePathTarget, ext, "_encoded.mkv", 1)
			}
			pathFile = filepath.ToSlash(pathFile)
			searchJobRequestChan <- &JobRequestResult{
				jobRequest: &model.JobRequest{
					SourcePath:      relativePathSource,
					DestinationPath: relativePathTarget,
					ForceCompleted:  parentJobRequest.ForceCompleted,
					ForceFailed:     parentJobRequest.ForceFailed,
					ForceExecuting:  parentJobRequest.ForceExecuting,
					ForceAdded:      parentJobRequest.ForceAdded,
					Priority:        parentJobRequest.Priority,
				},
				errors: jobRequestErrors,
			}
		}
		return nil
	})
}

type ScheduleJobRequestResult struct {
	ScheduledJobs    []*model.Video           `json:"scheduled"`
	FailedJobRequest []*model.JobRequestError `json:"failed"`
	SkippedFiles     []*model.JobRequestError `json:"skipped"`
}

func (R *RuntimeScheduler) scheduleJobRequest(ctx context.Context, jobRequest *model.JobRequest) (video *model.Video, err error) {
	priority := jobRequest.Priority
	err = R.repo.WithTransaction(ctx, func(ctx context.Context, tx repository.Repository) error {
		video, err = tx.GetJobByPath(ctx, jobRequest.SourcePath)
		if err != nil {
			return err
		}
		var eventsToAdd []*model.TaskEvent
		if video == nil {
			newUUID, _ := uuid.NewUUID()
			video = &model.Video{
				SourcePath:      jobRequest.SourcePath,
				DestinationPath: jobRequest.DestinationPath,
				Id:              newUUID,
			}
			err = tx.AddVideo(ctx, video)
			if err != nil {
				return err
			}
			startEvent := video.AddEvent(model.NotificationEvent, model.JobNotification, model.AddedNotificationStatus)
			eventsToAdd = append(eventsToAdd, startEvent)
		} else {
			//If video exist we check if we can retry the job
			lastEvent := video.Events.GetLatestPerNotificationType(model.JobNotification)
			status := video.Events.GetStatus()
			if jobRequest.ForceExecuting && status == model.StartedNotificationStatus {
				cancelEvent := video.AddEvent(model.NotificationEvent, model.JobNotification, model.CanceledNotificationStatus)
				eventsToAdd = append(eventsToAdd, cancelEvent)
			}
			if (jobRequest.ForceCompleted && status == model.CompletedNotificationStatus) ||
				(jobRequest.ForceFailed && (status == model.FailedNotificationStatus || status == model.CanceledNotificationStatus)) ||
				(jobRequest.ForceAdded && (status == model.AddedNotificationStatus || status == model.ReAddedNotificationStatus)) ||
				(jobRequest.ForceExecuting && status == model.StartedNotificationStatus) {
				requeueEvent := video.AddEvent(model.NotificationEvent, model.JobNotification, model.ReAddedNotificationStatus)
				eventsToAdd = append(eventsToAdd, requeueEvent)
			} else if !(jobRequest.ForceExecuting && status == model.StartedNotificationStatus) {
				return errors.New(fmt.Sprintf("%s (%s) job is in %s state by %s, can not be rescheduled", video.Id.String(), jobRequest.SourcePath, lastEvent.Status, lastEvent.WorkerName))
			}
		}
		if len(eventsToAdd) > 0 {
			for _, taskEvent := range eventsToAdd {
				err = tx.AddNewTaskEvent(ctx, taskEvent)
				if err != nil {
					return err
				}
			}
		}
		if priority == 0 {
			f, err := os.Open(filepath.Join(R.config.DownloadPath, jobRequest.SourcePath))
			if err != nil {
				return err
			}
			defer f.Close()
			data, err := ffprobe.ProbeReader(ctx, f)
			if err != nil {
				return err
			}
			if data.Format.DurationSeconds < 1800 { //30Min
				priority = 1
			} else if data.Format.DurationSeconds < 3600 { //60 Min
				priority = 2
			} else if data.Format.DurationSeconds < 7200 { //2h
				priority = 3
			} else if data.Format.DurationSeconds < 10800 { //3h
				priority = 4
			} else if data.Format.DurationSeconds > 10800 { //+3h
				priority = 5
			}
		}

		downloadURL, _ := url.Parse(fmt.Sprintf("%s/api/v1/download?uuid=%s", R.config.Domain.String(), video.Id.String()))
		uploadURL, _ := url.Parse(fmt.Sprintf("%s/api/v1/upload?uuid=%s", R.config.Domain.String(), video.Id.String()))
		checksumURL, _ := url.Parse(fmt.Sprintf("%s/api/v1/checksum?uuid=%s", R.config.Domain.String(), video.Id.String()))
		task := &model.TaskEncode{
			Id:          video.Id,
			DownloadURL: downloadURL.String(),
			UploadURL:   uploadURL.String(),
			ChecksumURL: checksumURL.String(),
			EventID:     video.Events.GetLatest().EventID,
			Priority:    priority,
		}
		return R.queue.PublishJobRequest(task)
	})
	return video, err
}

func (R *RuntimeScheduler) ScheduleJobRequests(ctx context.Context, jobRequest *model.JobRequest) (result *ScheduleJobRequestResult, returnError error) {
	result = &ScheduleJobRequestResult{}
	searchJobRequestChan := make(chan *JobRequestResult, 10)
	_, returnError = os.Stat(filepath.Join(R.config.DownloadPath, jobRequest.SourcePath))
	if os.IsNotExist(returnError) {
		return nil, returnError
	}

	go R.createNewJobRequestByJobRequestDirectory(ctx, jobRequest, searchJobRequestChan)

	for jobRequestResponse := range searchJobRequestChan {
		var err error
		var video *model.Video
		if jobRequestResponse.errors == nil {
			video, err = R.scheduleJobRequest(ctx, jobRequestResponse.jobRequest)
			if err == nil {
				video.Events = nil
			}
		} else {
			b, _ := json.Marshal(jobRequestResponse.errors)
			err = errors.New(string(b))
		}
		if err != nil {
			if errors.Is(err, ErrorFileSkipped) {
				result.SkippedFiles = append(result.SkippedFiles, &model.JobRequestError{
					JobRequest: *jobRequestResponse.jobRequest,
					Error:      errors.Unwrap(err).Error(),
				})
			} else {
				result.FailedJobRequest = append(result.FailedJobRequest, &model.JobRequestError{
					JobRequest: *jobRequestResponse.jobRequest,
					Error:      err.Error(),
				})
			}
		} else {
			result.ScheduledJobs = append(result.ScheduledJobs, video)
		}
	}
	return result, returnError
}

func (R *RuntimeScheduler) CancelJob(ctx context.Context, uuid string) error {
	video, err := R.repo.GetJob(ctx, uuid)
	if err != nil {
		if errors.Is(err, repository.ElementNotFound) {
			return ErrorJobNotFound
		}
		return err
	}
	lastEvent := video.Events.GetLatestPerNotificationType(model.JobNotification)
	status := lastEvent.Status
	if status == model.StartedNotificationStatus {
		jobAction := &model.JobEvent{
			Id:     video.Id,
			Action: model.CancelJob,
		}

		worker, err := R.repo.GetWorker(ctx, lastEvent.WorkerName)
		if err != nil {
			if errors.Is(err, repository.ElementNotFound) {
				return ErrorJobNotFound
			}
			return err
		}
		R.queue.PublishJobEvent(jobAction, worker.QueueName)
	} else {
		return fmt.Errorf("%w: job in status %s", ErrorInvalidStatus, status)
	}
	return nil
}

func (R *RuntimeScheduler) isValidStremeableJob(ctx context.Context, uuid string) (*model.Video, error) {
	video, err := R.repo.GetJob(ctx, uuid)
	if err != nil {
		return nil, err
	}
	status := video.Events.GetLatestPerNotificationType(model.JobNotification).Status
	if status != model.StartedNotificationStatus {
		return nil, fmt.Errorf("%w: job is in status %s", ErrorStreamNotAllowed, status)
	}
	return video, nil
}
func (R *RuntimeScheduler) GetDownloadJobWriter(ctx context.Context, uuid string) (*DownloadJobStream, error) {
	video, err := R.isValidStremeableJob(ctx, uuid)
	if err != nil {
		return nil, err
	}
	filePath := filepath.Join(R.config.DownloadPath, video.SourcePath)
	downloadFile, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrorJobNotFound
		} else {
			return nil, err
		}
	}
	dfStat, err := downloadFile.Stat()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrorJobNotFound
		} else {
			return nil, err
		}
	}
	return &DownloadJobStream{
		JobStream: &JobStream{
			video:             video,
			file:              downloadFile,
			path:              filePath,
			checksumPublisher: R.checksumChan,
		},
		FileSize: dfStat.Size(),
		FileName: dfStat.Name(),
	}, nil

}

func (R *RuntimeScheduler) GetUploadJobWriter(ctx context.Context, uuid string) (*UploadJobStream, error) {
	video, err := R.isValidStremeableJob(ctx, uuid)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(R.config.UploadPath, video.DestinationPath)
	err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return nil, err
	}
	temporalPath := filePath + ".upload"
	uploadFile, err := os.OpenFile(temporalPath, os.O_TRUNC|os.O_CREATE|os.O_RDWR, os.ModePerm)
	return &UploadJobStream{
		&JobStream{
			video:        video,
			file:         uploadFile,
			path:         filePath,
			temporalPath: temporalPath,
		},
	}, nil
}

func (R *RuntimeScheduler) GetChecksum(ctx context.Context, uuid string) (string, error) {
	video, err := R.repo.GetJob(ctx, uuid)
	if err != nil {
		return "", err
	}
	filePath := filepath.Join(R.config.DownloadPath, video.SourcePath)
	checksum := R.pathChecksumMap[filePath]
	if checksum == "" {
		return "", fmt.Errorf("%w: Checksum not found for %s", ErrorJobNotFound, filePath)
	}
	return checksum, nil
}

func (S *RuntimeScheduler) stop() {

}

/*
	func init() {
		f, _ := os.Open("/mnt/d/encode_public_videos.csv")
		fileScanner := bufio.NewScanner(f)
		fileScanner.Split(bufio.ScanLines)
		i := 0
		for fileScanner.Scan() {
			i++
			line := fileScanner.Text()
			if strings.Contains(line, "265") {
				continue
			}
			if strings.Contains(line, "[ ]") {
				continue
			}
			if !x264ex.MatchString(line) {
				fmt.Printf("264: %d FAIL on %s\n\r", i, line)
			}

			if strings.Contains(strings.ToLower(line), "aac") {
				continue
			}
			if !ac3ex.MatchString(line) {
				fmt.Printf("AC3: %d FAIL on %s\n\r", i, line)
			}
			formatTargetName(line)

		}
	}
*/
func formatTargetName(path string) string {
	p := x264ex.ReplaceAllString(path, "x265")
	p = ac3ex.ReplaceAllString(p, "AAC")
	extension := filepath.Ext(p)
	p = strings.Replace(p, extension, ".mkv", 1)

	return p
}

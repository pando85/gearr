package scheduler

import (
	"context"
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
)

var (
	x264ex = regexp.MustCompile(`(?i)(((x|h)264)|mpeg-4|mpeg-1|mpeg-2|mpeg|xvid|divx|vc-1|av1|vp8|vp9|wmv3|mp43)`)
	ac3ex  = regexp.MustCompile(`(?i)(ac3|eac3|pcm|flac|mp2|dts|mp2|mp3|truehd|wma|vorbis|opus|mpeg audio)`)
)

type Scheduler interface {
	Run(wg *sync.WaitGroup, ctx context.Context)
	ScheduleJobRequest(ctx context.Context, jobRequest *model.JobRequest) (*model.Job, error)
	GetJob(ctx context.Context, uuid string) (*model.Job, error)
	DeleteJob(ctx context.Context, uuid string) error
	GetJobs(ctx context.Context) (*[]model.Job, error)
	GetUploadJobWriter(ctx context.Context, uuid string) (*UploadJobStream, error)
	GetDownloadJobWriter(ctx context.Context, uuid string) (*DownloadJobStream, error)
	GetChecksum(ctx context.Context, uuid string) (string, error)
	GetWorkers(ctx context.Context) (*[]model.Worker, error)
	GetUpdateJobsChan(ctx context.Context) (uuid.UUID, chan *model.JobUpdateNotification)
	CloseUpdateJobsChan(id uuid.UUID)
}

type SchedulerConfig struct {
	ScheduleTime time.Duration `mapstructure:"scheduleTime"`
	JobTimeout   time.Duration `mapstructure:"jobTimeout"`
	DownloadPath string        `mapstructure:"downloadPath"`
	UploadPath   string        `mapstructure:"uploadPath"`
	Domain       *url.URL
	MinFileSize  int64 `mapstructure:"minFileSize"`
}

type RuntimeScheduler struct {
	config             SchedulerConfig
	repo               repository.Repository
	queue              queue.BrokerServer
	checksumChan       chan PathChecksum
	updateJobsChannels map[uuid.UUID]chan *model.JobUpdateNotification
	jobChannelsMutex   sync.Mutex
	pathChecksumMap    map[string]string
}

func NewScheduler(config SchedulerConfig, repo repository.Repository, queue queue.BrokerServer) (*RuntimeScheduler, error) {
	runtimeScheduler := &RuntimeScheduler{
		config:             config,
		repo:               repo,
		queue:              queue,
		checksumChan:       make(chan PathChecksum),
		updateJobsChannels: make(map[uuid.UUID]chan *model.JobUpdateNotification, 0),
		pathChecksumMap:    make(map[string]string),
	}

	return runtimeScheduler, nil
}

func (R *RuntimeScheduler) Run(wg *sync.WaitGroup, ctx context.Context) {
	log.Info("starting scheduler")
	R.start(ctx)
	log.Info("progressing scheduler")
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

func (R *RuntimeScheduler) GetUpdateJobsChan(ctx context.Context) (uuid.UUID, chan *model.JobUpdateNotification) {
	ch := make(chan *model.JobUpdateNotification)
	id := uuid.New()
	R.jobChannelsMutex.Lock()
	R.updateJobsChannels[id] = ch
	R.jobChannelsMutex.Unlock()
	return id, ch
}

func (R *RuntimeScheduler) CloseUpdateJobsChan(id uuid.UUID) {
	R.jobChannelsMutex.Lock()
	delete(R.updateJobsChannels, id)
	R.jobChannelsMutex.Unlock()
}

func (R *RuntimeScheduler) sendUpdateJobsNotification(notification *model.JobUpdateNotification) {
	for _, ch := range R.updateJobsChannels {
		ch <- notification
	}
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

			if jobEvent.EventType != model.PingEvent {
				jobUpdateNotification := model.JobUpdateNotification{
					Id:        jobEvent.Id,
					Status:    jobEvent.Status,
					Message:   jobEvent.Message,
					EventTime: jobEvent.EventTime,
				}
				R.sendUpdateJobsNotification(&jobUpdateNotification)
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
				if taskEvent.Status == model.ProgressingNotificationStatus {
					log.Infof("rescheduling %s after job timeout", taskEvent.Id.String())
					video, err := R.repo.GetJob(ctx, taskEvent.Id.String())
					if err != nil {
						log.Error(err)
						continue
					}
					jobRequest := &model.JobRequest{
						SourcePath:      video.SourcePath,
						DestinationPath: video.DestinationPath,
					}
					_, err = R.scheduleJobRequest(ctx, jobRequest)
					if err != nil {
						log.Error(err)
					}
				}
			}
		}
	}
}

func (R *RuntimeScheduler) scheduleJobRequest(ctx context.Context, jobRequest *model.JobRequest) (video *model.Job, err error) {
	err = R.repo.WithTransaction(ctx, func(ctx context.Context, tx repository.Repository) error {
		video, err = tx.GetJobByPath(ctx, jobRequest.SourcePath)
		if err != nil {
			return err
		}
		var eventsToAdd []*model.TaskEvent
		if video != nil {
			return &model.CustomError{Message: "job already exists"}
		}
		newUUID, _ := uuid.NewUUID()
		video = &model.Job{
			SourcePath:      jobRequest.SourcePath,
			DestinationPath: jobRequest.DestinationPath,
			Id:              newUUID,
		}
		err = tx.AddJob(ctx, video)
		if err != nil {
			return err
		}
		startEvent := video.AddEvent(model.NotificationEvent, model.JobNotification, model.QueuedNotificationStatus)
		eventsToAdd = append(eventsToAdd, startEvent)
		if len(eventsToAdd) > 0 {
			for _, taskEvent := range eventsToAdd {
				err = tx.AddNewTaskEvent(ctx, taskEvent)
				if err != nil {
					return err
				}
			}
		}

		downloadURL, _ := url.Parse(fmt.Sprintf("%s/api/v1/job/%s/download", R.config.Domain.String(), video.Id.String()))
		uploadURL, _ := url.Parse(fmt.Sprintf("%s/api/v1/job/%s/upload", R.config.Domain.String(), video.Id.String()))
		checksumURL, _ := url.Parse(fmt.Sprintf("%s/api/v1/job/%s/checksum", R.config.Domain.String(), video.Id.String()))
		task := &model.TaskEncode{
			Id:          video.Id,
			DownloadURL: downloadURL.String(),
			UploadURL:   uploadURL.String(),
			ChecksumURL: checksumURL.String(),
			EventID:     video.Events.GetLatest().EventID,
		}
		return R.queue.PublishJobRequest(task)
	})
	return video, err
}

func (R *RuntimeScheduler) ScheduleJobRequest(ctx context.Context, jobRequest *model.JobRequest) (*model.Job, error) {
	filePath := filepath.Join(R.config.DownloadPath, jobRequest.SourcePath)
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, err
	}

	if fileInfo.IsDir() {
		errorMessage := fmt.Sprintf("%s is a directory", filePath)
		return nil, &model.CustomError{Message: errorMessage}
	}

	if fileInfo.Size() < R.config.MinFileSize {
		errorMessage := fmt.Sprintf("%s File size must be bigger than %d", filePath, R.config.MinFileSize)
		return nil, &model.CustomError{Message: errorMessage}
	}
	extension := filepath.Ext(fileInfo.Name())[1:]
	if !helper.ValidExtension(extension) {
		errorMessage := fmt.Sprintf("%s Invalid Extension %s", filePath, extension)
		return nil, &model.CustomError{Message: errorMessage}
	}

	relativePathSource, err := filepath.Rel(R.config.DownloadPath, filepath.FromSlash(filePath))
	if err != nil {
		errorMessage := fmt.Sprintf("%s is not relative download path", filePath)
		return nil, &model.CustomError{Message: errorMessage}
	}

	relativePathTarget := formatTargetName(relativePathSource)
	if relativePathTarget == relativePathSource {
		ext := filepath.Ext(relativePathTarget)
		relativePathTarget = strings.Replace(relativePathTarget, ext, "_encoded.mkv", 1)
	}

	filteredJobRequest := &model.JobRequest{
		SourcePath:      relativePathSource,
		DestinationPath: relativePathTarget,
	}

	video, err := R.scheduleJobRequest(ctx, filteredJobRequest)
	if err != nil {
		return nil, err
	}

	jobUpdateNotification := model.JobUpdateNotification{
		Id:              video.Id,
		SourcePath:      video.SourcePath,
		DestinationPath: video.DestinationPath,
	}

	R.sendUpdateJobsNotification(&jobUpdateNotification)
	return video, nil
}

func (R *RuntimeScheduler) GetJob(ctx context.Context, uuid string) (*model.Job, error) {
	return R.repo.GetJob(ctx, uuid)
}

func (R *RuntimeScheduler) DeleteJob(ctx context.Context, uuid string) error {
	return R.repo.DeleteJob(ctx, uuid)
}

func (R *RuntimeScheduler) GetJobs(ctx context.Context) (*[]model.Job, error) {
	return R.repo.GetJobs(ctx)
}

func (R *RuntimeScheduler) isValidStremeableJob(ctx context.Context, uuid string) (*model.Job, error) {
	video, err := R.repo.GetJob(ctx, uuid)
	if err != nil {
		return nil, err
	}
	status := video.Events.GetLatestPerNotificationType(model.JobNotification).Status
	if status != model.ProgressingNotificationStatus {
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
	}, err
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

func (R *RuntimeScheduler) GetWorkers(ctx context.Context) (*[]model.Worker, error) {
	return R.repo.GetWorkers(ctx)
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

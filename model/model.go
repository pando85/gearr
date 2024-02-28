package model

import (
	"gearr/helper/max"
	"os"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type EventType string
type NotificationType string
type NotificationStatus string
type JobAction string
type TaskEvents []*TaskEvent

type CustomError struct {
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}

const (
	PingEvent         EventType = "Ping"
	NotificationEvent EventType = "Notification"

	JobNotification        NotificationType = "Job"
	DownloadNotification   NotificationType = "Download"
	UploadNotification     NotificationType = "Upload"
	MKVExtractNotification NotificationType = "MKVExtract"
	FFProbeNotification    NotificationType = "FFProbe"
	PGSNotification        NotificationType = "PGS"
	FFMPEGSNotification    NotificationType = "FFMPEG"

	QueuedNotificationStatus      NotificationStatus = "queued"
	ReQueuedNotificationStatus    NotificationStatus = "requeued"
	ProgressingNotificationStatus NotificationStatus = "progressing"
	CompletedNotificationStatus   NotificationStatus = "completed"
	CanceledNotificationStatus    NotificationStatus = "canceled"
	FailedNotificationStatus      NotificationStatus = "failed"

	EncodeJobType   JobType = "encode"
	PGSToSrtJobType JobType = "pgstosrt"
)

type Identity interface {
	getUUID() uuid.UUID
}
type Job struct {
	SourcePath      string           `json:"source_path,omitempty"`
	DestinationPath string           `json:"destination_path,omitempty"`
	Id              uuid.UUID        `json:"id"`
	Events          TaskEvents       `json:"events,omitempty"`
	Status          string           `json:"status,omitempty"`
	StatusPhase     NotificationType `json:"status_phase,omitempty"`
	StatusMessage   string           `json:"status_message,omitempty"`
	LastUpdate      *time.Time       `json:"last_update,omitempty"`
}

type JobEventQueue struct {
	Queue    string
	JobEvent *JobEvent
}
type Worker struct {
	Name      string    `json:"name"`
	Ip        string    `json:"id"`
	QueueName string    `json:"queue_name"`
	LastSeen  time.Time `json:"last_seen"`
}

type ControlEvent struct {
	Event       *TaskEncode
	ControlChan chan interface{}
}

type JobEvent struct {
	Id     uuid.UUID `json:"id"`
	Action JobAction `json:"action"`
}

type JobType string

type TaskEncode struct {
	Id          uuid.UUID `json:"id"`
	DownloadURL string    `json:"downloadURL"`
	UploadURL   string    `json:"uploadURL"`
	ChecksumURL string    `json:"checksumURL"`
	EventID     int       `json:"eventID"`
}

type WorkTaskEncode struct {
	TaskEncode     *TaskEncode
	WorkDir        string
	SourceFilePath string
	TargetFilePath string
}

type TaskPGS struct {
	Id          uuid.UUID `json:"id"`
	PGSID       int       `json:"pgsid"`
	PGSdata     []byte    `json:"pgsdata"`
	PGSLanguage string    `json:"pgslanguage"`
	ReplyTo     string    `json:"replyto"`
}

type TaskPGSResponse struct {
	Id    uuid.UUID `json:"id"`
	PGSID int       `json:"pgsid"`
	Srt   []byte    `json:"srt"`
	Err   string    `json:"error"`
	Queue string    `json:"queue"`
}

func (V TaskEncode) getUUID() uuid.UUID {
	return V.Id
}
func (V TaskPGS) getUUID() uuid.UUID {
	return V.Id
}

type JobUpdateNotification struct {
	Id              uuid.UUID          `json:"id"`
	Status          NotificationStatus `json:"status"`
	StatusPhase     NotificationType   `json:"status_phase"`
	Message         string             `json:"message"`
	EventTime       time.Time          `json:"event_time"`
	SourcePath      string             `json:"source_path,omitempty"`
	DestinationPath string             `json:"destination_path,omitempty"`
}

type TaskEvent struct {
	Id               uuid.UUID          `json:"id"`
	EventID          int                `json:"event_id"`
	EventType        EventType          `json:"event_type"`
	WorkerName       string             `json:"worker_name"`
	WorkerQueue      string             `json:"worker_queue"`
	EventTime        time.Time          `json:"event_time"`
	IP               string             `json:"ip"`
	NotificationType NotificationType   `json:"notification_type"`
	Status           NotificationStatus `json:"status"`
	Message          string             `json:"message"`
}

type TaskStatus struct {
	LastState *TaskEvent
	Task      *WorkTaskEncode
}

func (e TaskEvent) IsDownloading() bool {
	if e.EventType != NotificationEvent {
		return false
	}
	if e.NotificationType == DownloadNotification && e.Status == ProgressingNotificationStatus {
		return true
	}

	if e.NotificationType == JobNotification && (e.Status == ProgressingNotificationStatus) {
		return true
	}
	return false
}

func (e TaskEvent) IsEncoding() bool {
	if e.EventType != NotificationEvent {
		return false
	}
	if e.NotificationType == DownloadNotification && e.Status == CompletedNotificationStatus {
		return true
	}

	if e.NotificationType == MKVExtractNotification && (e.Status == ProgressingNotificationStatus || e.Status == CompletedNotificationStatus) {
		return true
	}
	if e.NotificationType == FFProbeNotification && (e.Status == ProgressingNotificationStatus || e.Status == CompletedNotificationStatus) {
		return true
	}
	if e.NotificationType == PGSNotification && (e.Status == ProgressingNotificationStatus || e.Status == CompletedNotificationStatus) {
		return true
	}
	if e.NotificationType == FFMPEGSNotification && e.Status == ProgressingNotificationStatus {
		return true
	}

	return false
}

func (e TaskEvent) IsUploading() bool {
	if e.EventType != NotificationEvent {
		return false
	}
	if e.NotificationType == FFMPEGSNotification && e.Status == CompletedNotificationStatus {
		return true
	}

	if e.NotificationType == UploadNotification && e.Status == ProgressingNotificationStatus {
		return true
	}

	return false
}

func (W *WorkTaskEncode) Clean() error {
	//log.Warnf("[%s] cleaning up task workspace", W.TaskEncode.Id.String())
	err := os.RemoveAll(W.WorkDir)
	if err != nil {
		return err
	}
	return nil
}

func (t *TaskEvents) GetLatest() *TaskEvent {
	if len(*t) == 0 {
		return nil
	}
	return max.Max(t).(*TaskEvent)
}
func (t *TaskEvents) GetLatestPerNotificationType(notificationType NotificationType) (returnEvent *TaskEvent) {
	log.Debugf("notification type: %+v", notificationType)

	if t == nil || len(*t) == 0 {
		log.Panic("task events are empty")
	}

	eventID := -1
	for _, event := range *t {
		if event.NotificationType == notificationType && event.EventID > eventID {
			eventID = event.EventID
			returnEvent = event
		}
	}
	return returnEvent
}
func (t *TaskEvents) GetStatus() NotificationStatus {
	return t.GetLatestPerNotificationType(JobNotification).Status
}

type JobRequest struct {
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
}

func (a TaskEvents) Len() int {
	return len(a)
}
func (a TaskEvents) Less(i, j int) bool {
	return a[i].EventID < a[j].EventID
}
func (a TaskEvents) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a TaskEvents) GetLastElement(i int) interface{} {
	return a[i]
}

func (v *Job) AddEvent(eventType EventType, notificationType NotificationType, notificationStatus NotificationStatus) (newEvent *TaskEvent) {
	latestEvent := v.Events.GetLatest()
	newEventID := 0
	if latestEvent != nil {
		newEventID = latestEvent.EventID + 1
	}

	newEvent = &TaskEvent{
		Id:               v.Id,
		EventID:          newEventID,
		EventType:        eventType,
		EventTime:        time.Now(),
		NotificationType: notificationType,
		Status:           notificationStatus,
	}
	v.Events = append(v.Events, newEvent)
	return newEvent
}

type Manager interface {
	EventNotification(event TaskEvent)
	ResponsePGSJob(response TaskPGSResponse) error
	RequestPGSJob(pgsJob TaskPGS) <-chan *TaskPGSResponse
}

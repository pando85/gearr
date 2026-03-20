package model

import (
	"errors"
	"gearr/helper"
	"gearr/helper/max"
	"os"
	"time"

	"github.com/google/uuid"
)

var (
	ErrJobExists = errors.New("job already exists")
)

type EventType string
type NotificationType string
type NotificationStatus string
type JobAction string
type TaskEvents []*TaskEvent

type CustomError struct {
	Message string
	Cause   error
}

func (e *CustomError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *CustomError) Unwrap() error {
	return e.Cause
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
	Priority        int              `json:"priority,omitempty"`
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
	helper.Debugf("notification type: %+v", notificationType)

	if t == nil || len(*t) == 0 {
		helper.Warnf("task events are empty for notification type %s", notificationType)
		return nil
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
	event := t.GetLatestPerNotificationType(JobNotification)
	if event == nil {
		return ""
	}
	return event.Status
}

type JobRequest struct {
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
	Priority        int    `json:"priority,omitempty"`
}

type TimeoutJob struct {
	Id              uuid.UUID          `json:"id"`
	SourcePath      string             `json:"source_path"`
	DestinationPath string             `json:"destination_path"`
	Status          NotificationStatus `json:"status"`
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
	EventNotification(event TaskEvent) error
	ResponsePGSJob(response TaskPGSResponse) error
	RequestPGSJob(pgsJob TaskPGS) <-chan *TaskPGSResponse
}

type BrokerClient interface {
	Manager
}

type FileProcessingSource string
type FileProcessingStatus string

const (
	ScannerSource FileProcessingSource = "scanner"
	WatcherSource FileProcessingSource = "watcher"

	QueuedStatus  FileProcessingStatus = "queued"
	SkippedStatus FileProcessingStatus = "skipped"
	InvalidStatus FileProcessingStatus = "invalid"
	X265Status    FileProcessingStatus = "x265"
	ErrorStatus   FileProcessingStatus = "error"
)

type FileProcessing struct {
	Id         int                  `json:"id"`
	Path       string               `json:"path"`
	DetectedAt time.Time            `json:"detected_at"`
	Source     FileProcessingSource `json:"source"`
	Status     FileProcessingStatus `json:"status"`
	Message    string               `json:"message,omitempty"`
	JobId      *uuid.UUID           `json:"job_id,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
}

type ScanStatus string

const (
	ScanRunning   ScanStatus = "running"
	ScanCompleted ScanStatus = "completed"
	ScanFailed    ScanStatus = "failed"
)

type LibraryScan struct {
	Id                string     `json:"id"`
	StartedAt         time.Time  `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	Status            ScanStatus `json:"status"`
	FilesFound        int        `json:"files_found"`
	FilesQueued       int        `json:"files_queued"`
	FilesSkippedSize  int        `json:"files_skipped_size"`
	FilesSkippedCodec int        `json:"files_skipped_codec"`
	FilesSkippedExist int        `json:"files_skipped_exists"`
	ErrorMessage      string     `json:"error_message,omitempty"`
}

type ScannedFile struct {
	Id            string    `json:"id"`
	FilePath      string    `json:"file_path"`
	FileSize      int64     `json:"file_size"`
	Codec         string    `json:"codec,omitempty"`
	LastScannedAt time.Time `json:"last_scanned_at"`
	Queued        bool      `json:"queued"`
	ScanId        string    `json:"scan_id,omitempty"`
}

type ScannerStatus struct {
	Enabled    bool          `json:"enabled"`
	IsScanning bool          `json:"is_scanning"`
	LastScan   *LibraryScan  `json:"last_scan,omitempty"`
	NextScanAt *time.Time    `json:"next_scan_at,omitempty"`
	Config     ScannerConfig `json:"config"`
}

type ScannerConfig struct {
	Enabled        bool          `json:"enabled"`
	Interval       time.Duration `json:"interval"`
	MinFileSize    int64         `json:"min_file_size"`
	Paths          []string      `json:"paths"`
	FileExtensions []string      `json:"file_extensions"`
}

type ScannerNotification struct {
	Type         string     `json:"type"`
	ScanId       string     `json:"scan_id"`
	Progress     int        `json:"progress"`
	FilesFound   int        `json:"files_found"`
	FilesQueued  int        `json:"files_queued"`
	FilesSkipped int        `json:"files_skipped"`
	CurrentPath  string     `json:"current_path,omitempty"`
	Status       ScanStatus `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

type WebhookProvider string

const (
	WebhookProviderRadarr  WebhookProvider = "radarr"
	WebhookProviderSonarr  WebhookProvider = "sonarr"
	WebhookProviderGeneric WebhookProvider = "generic"
)

type WebhookAuthConfig struct {
	APIKey string `json:"-" mapstructure:"apiKey"`
}

type WebhookConfig struct {
	Enabled   bool              `json:"enabled" mapstructure:"enabled"`
	Radarr    WebhookAuthConfig `json:"radarr" mapstructure:"radarr"`
	Sonarr    WebhookAuthConfig `json:"sonarr" mapstructure:"sonarr"`
	Providers map[string]string `json:"providers,omitempty" mapstructure:"providers"`
}

func (c *WebhookAuthConfig) IsValid() bool {
	return c.APIKey != ""
}

func (c *WebhookAuthConfig) ValidateAPIKey(key string) bool {
	if c.APIKey == "" {
		return false
	}
	return c.APIKey == key
}

func NewWebhookConfig() WebhookConfig {
	return WebhookConfig{
		Enabled:   false,
		Providers: make(map[string]string),
	}
}

func (c *WebhookConfig) AddProvider(name string, provider WebhookProvider, apiKey string) {
	c.Enabled = true
}

func (c *WebhookConfig) GetProvider(name string) *WebhookAuthConfig {
	switch name {
	case string(WebhookProviderRadarr):
		if c.Radarr.APIKey != "" {
			return &WebhookAuthConfig{APIKey: c.Radarr.APIKey}
		}
	case string(WebhookProviderSonarr):
		if c.Sonarr.APIKey != "" {
			return &WebhookAuthConfig{APIKey: c.Sonarr.APIKey}
		}
	default:
		if apiKey, exists := c.Providers[name]; exists && apiKey != "" {
			return &WebhookAuthConfig{APIKey: apiKey}
		}
	}
	return nil
}

func (c *WebhookConfig) ValidateAuth(providerName string, apiKey string) bool {
	provider := c.GetProvider(providerName)
	if provider == nil {
		return false
	}
	return provider.ValidateAPIKey(apiKey)
}

type PriorityLevel string

const (
	PriorityLow    PriorityLevel = "low"
	PriorityNormal PriorityLevel = "normal"
	PriorityHigh   PriorityLevel = "high"
	PriorityUrgent PriorityLevel = "urgent"
)

type PriorityRuleType string

const (
	PriorityBySize        PriorityRuleType = "size"
	PriorityByAge         PriorityRuleType = "age"
	PriorityByPathPattern PriorityRuleType = "path_pattern"
)

type PriorityRule struct {
	Type      PriorityRuleType `mapstructure:"type" json:"type"`
	Threshold int64            `mapstructure:"threshold" json:"threshold,omitempty"`
	Pattern   string           `mapstructure:"pattern" json:"pattern,omitempty"`
	Level     PriorityLevel    `mapstructure:"level" json:"level"`
}

type PriorityConfig struct {
	Enabled         bool           `mapstructure:"enabled" json:"enabled"`
	DefaultPriority PriorityLevel  `mapstructure:"defaultPriority" json:"default_priority"`
	SizeThresholds  SizeThresholds `mapstructure:"sizeThresholds" json:"size_thresholds"`
	AgeThresholds   AgeThresholds  `mapstructure:"ageThresholds" json:"age_thresholds"`
	CustomRules     []PriorityRule `mapstructure:"customRules" json:"custom_rules,omitempty"`
}

type SizeThresholds struct {
	LargeFileSizeMB int64         `mapstructure:"largeFileSizeMB" json:"large_file_size_mb"`
	SmallFileSizeMB int64         `mapstructure:"smallFileSizeMB" json:"small_file_size_mb"`
	LargeFileLevel  PriorityLevel `mapstructure:"largeFileLevel" json:"large_file_level"`
	SmallFileLevel  PriorityLevel `mapstructure:"smallFileLevel" json:"small_file_level"`
}

type AgeThresholds struct {
	OldFileHours    int           `mapstructure:"oldFileHours" json:"old_file_hours"`
	RecentFileHours int           `mapstructure:"recentFileHours" json:"recent_file_hours"`
	OldFileLevel    PriorityLevel `mapstructure:"oldFileLevel" json:"old_file_level"`
	RecentFileLevel PriorityLevel `mapstructure:"recentFileLevel" json:"recent_file_level"`
}

type WebhookEventStatus string

const (
	WebhookEventStatusSuccess WebhookEventStatus = "success"
	WebhookEventStatusFailed  WebhookEventStatus = "failed"
	WebhookEventStatusSkipped WebhookEventStatus = "skipped"
)

type WebhookEvent struct {
	Id           int64              `json:"id"`
	Source       WebhookProvider    `json:"source"`
	EventType    string             `json:"event_type"`
	FilePath     string             `json:"file_path,omitempty"`
	Status       WebhookEventStatus `json:"status"`
	Message      string             `json:"message,omitempty"`
	Payload      string             `json:"payload,omitempty"`
	JobId        *uuid.UUID         `json:"job_id,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	ErrorDetails string             `json:"error_details,omitempty"`
}

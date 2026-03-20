package repository

import (
	"context"
	"database/sql"
	"gearr/model"
	"time"
)

type JobRepository interface {
	GetJob(ctx context.Context, uuid string) (*model.Job, error)
	DeleteJob(ctx context.Context, uuid string) error
	GetJobs(ctx context.Context) (*[]model.Job, error)
	GetJobsWithOptions(ctx context.Context, sortBy string, priority *int) (*[]model.Job, error)
	GetJobByPath(ctx context.Context, path string) (*model.Job, error)
	AddJob(ctx context.Context, job *model.Job) error
	UpdateJobPriority(ctx context.Context, jobID string, priority int) error
}

type WorkerRepository interface {
	GetWorker(ctx context.Context, name string) (*model.Worker, error)
	GetWorkers(ctx context.Context) (*[]model.Worker, error)
	PingServerUpdate(ctx context.Context, name string, ip string, queueName string) error
}

type QueueRepository interface {
	EnqueueEncodeJob(ctx context.Context, task *model.TaskEncode) error
	DequeueEncodeJob(ctx context.Context, workerName string) (*model.TaskEncode, error)
	EnqueuePGSJob(ctx context.Context, pgs *model.TaskPGS) error
	DequeuePGSJob(ctx context.Context, workerName string) (*model.TaskPGS, error)
	EnqueuePGSResponse(ctx context.Context, resp *model.TaskPGSResponse) error
	DequeuePGSResponse(ctx context.Context, replyToQueue string) (*model.TaskPGSResponse, error)
	EnqueueTaskEvent(ctx context.Context, event *model.TaskEvent) error
	DequeueTaskEvents(ctx context.Context, limit int) ([]*model.TaskEvent, error)
	EnqueueJobAction(ctx context.Context, jobID string, workerName string, action model.JobAction) error
	DequeueJobActions(ctx context.Context, workerName string) ([]*model.JobEvent, error)
}

type EventRepository interface {
	ProcessEvent(ctx context.Context, event *model.TaskEvent) error
	AddNewTaskEvent(ctx context.Context, event *model.TaskEvent) error
	GetTimeoutJobs(ctx context.Context, timeout time.Duration) ([]*model.TimeoutJob, error)
}

type ScanRepository interface {
	AddFileProcessing(ctx context.Context, fp *model.FileProcessing) error
	GetFileProcessingByPath(ctx context.Context, path string) (*model.FileProcessing, error)
	GetRecentFileProcessings(ctx context.Context, limit int, source model.FileProcessingSource) ([]*model.FileProcessing, error)
	CreateScan(ctx context.Context, scan *model.LibraryScan) error
	UpdateScan(ctx context.Context, scan *model.LibraryScan) error
	GetScan(ctx context.Context, id string) (*model.LibraryScan, error)
	GetLatestScan(ctx context.Context) (*model.LibraryScan, error)
	GetScanHistory(ctx context.Context, limit int) ([]*model.LibraryScan, error)
	UpsertScannedFile(ctx context.Context, file *model.ScannedFile) error
	GetScannedFile(ctx context.Context, path string) (*model.ScannedFile, error)
	GetScannedFilesByScan(ctx context.Context, scanID string) ([]*model.ScannedFile, error)
}

type BaseRepository interface {
	Initialize(ctx context.Context) error
	WithTransaction(ctx context.Context, transactionFunc func(ctx context.Context, tx Repository) error) error
	GetDB() *sql.DB
}

type Repository interface {
	BaseRepository
	JobRepository
	WorkerRepository
	QueueRepository
	EventRepository
	ScanRepository
	getConnection(ctx context.Context) (Transaction, error)
}

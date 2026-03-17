package repository

import (
	"context"
	"database/sql"
	"fmt"
	"gearr/internal/constants"
	"gearr/model"
	"strings"
	"time"

	_ "embed"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
)

var (
	ErrElementNotFound = fmt.Errorf("element not found")
)

type Repository interface {
	getConnection(ctx context.Context) (Transaction, error)
	Initialize(ctx context.Context) error
	ProcessEvent(ctx context.Context, event *model.TaskEvent) error
	PingServerUpdate(ctx context.Context, name string, ip string, queueName string) error
	GetTimeoutJobs(ctx context.Context, timeout time.Duration) ([]*model.TaskEvent, error)
	GetJob(ctx context.Context, uuid string) (*model.Job, error)
	DeleteJob(ctx context.Context, uuid string) error
	GetJobs(ctx context.Context) (*[]model.Job, error)
	GetJobByPath(ctx context.Context, path string) (*model.Job, error)
	AddNewTaskEvent(ctx context.Context, event *model.TaskEvent) error
	AddJob(ctx context.Context, job *model.Job) error
	WithTransaction(ctx context.Context, transactionFunc func(ctx context.Context, tx Repository) error) error
	GetWorker(ctx context.Context, name string) (*model.Worker, error)
	GetWorkers(ctx context.Context) (*[]model.Worker, error)
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
	GetDB() *sql.DB
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

type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type SQLTransaction struct {
	tx *sql.Tx
}

func (S *SQLTransaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return S.tx.Exec(query, args...)

}

func (S *SQLTransaction) Prepare(query string) (*sql.Stmt, error) {
	return S.tx.Prepare(query)
}

func (S *SQLTransaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return S.tx.Query(query, args...)
}

func (S *SQLTransaction) QueryRow(query string, args ...interface{}) *sql.Row {
	return S.tx.QueryRow(query, args...)
}

func (S *SQLTransaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return S.tx.QueryContext(ctx, query, args...)
}

func (S *SQLTransaction) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return S.tx.ExecContext(ctx, query, args...)
}

func (S *SQLTransaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return S.tx.QueryRowContext(ctx, query, args...)
}

type SQLRepository struct {
	db  *sql.DB
	con Transaction
}

type SQLServerConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	Driver   string `mapstructure:"driver"`
	SSLMode  string `mapstructure:"sslmode"`
}

func NewSQLRepository(config SQLServerConfig) (*SQLRepository, error) {
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&default_query_exec_mode=simple_protocol", config.User, config.Password, config.Host, config.Port, config.Database, config.SSLMode)
	db, err := sql.Open(config.Driver, connectionString)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	db.SetMaxOpenConns(constants.DBMaxOpenConns)
	db.SetConnMaxLifetime(constants.DBConnMaxLifetime)
	db.SetMaxIdleConns(constants.DBMaxIdleConns)
	/*	go func(){
		for {
			fmt.Printf("In use %d not use %d  open %d wait %d\n",db.Stats().Idle, db.Stats().InUse, db.Stats().OpenConnections,db.Stats().WaitCount)
			time.Sleep(time.Second*5)
		}
	}()*/
	return &SQLRepository{
		db: db,
	}, nil

}

func (S *SQLRepository) Initialize(ctx context.Context) error {
	return S.prepareDatabase(ctx)
}

func (S *SQLRepository) ProcessEvent(ctx context.Context, taskEvent *model.TaskEvent) error {
	var err error
	switch taskEvent.EventType {
	case model.PingEvent:
		err = S.PingServerUpdate(ctx, taskEvent.WorkerName, taskEvent.WorkerQueue, taskEvent.IP)
	case model.NotificationEvent:
		err = S.AddNewTaskEvent(ctx, taskEvent)
		/*if taskEvent.NotificationType == model.FFProbeNotification && taskEvent.Status ==  model.CompletedNotificationStatus {
			taskEvent.
		}*/
	}
	return err
}

//go:embed resources/database.sql
var databaseScript string

func (S *SQLRepository) prepareDatabase(ctx context.Context) (returnError error) {
	err := S.WithTransaction(ctx, func(ctx context.Context, tx Repository) error {
		con, err := tx.getConnection(ctx)
		if err != nil {
			return err
		}
		log.Debug("prepare database")
		_, err = con.ExecContext(ctx, databaseScript)
		return err
	})
	return err
}

func (S *SQLRepository) getConnection(ctx context.Context) (Transaction, error) {
	//return S.db.Conn(ctx)
	if S.con != nil {
		return S.con, nil
	}
	return S.db, nil
}
func (S *SQLRepository) GetWorker(ctx context.Context, name string) (worker *model.Worker, err error) {
	db, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	return S.getWorker(ctx, db, name)
}

func (S *SQLRepository) getWorker(ctx context.Context, db Transaction, name string) (*model.Worker, error) {
	rows, err := db.QueryContext(ctx, "SELECT * FROM workers WHERE name=$1", name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	worker := model.Worker{}
	found := false
	if rows.Next() {
		if err := rows.Scan(&worker.Name, &worker.Ip, &worker.QueueName, &worker.LastSeen); err != nil {
			return nil, err
		}
		found = true
	}
	if !found {
		return nil, fmt.Errorf("%w, %s", ErrElementNotFound, name)
	}
	return &worker, nil
}

func (S *SQLRepository) GetWorkers(ctx context.Context) (*[]model.Worker, error) {
	db, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	return S.getWorkers(ctx, db)
}

func (S *SQLRepository) getWorkers(ctx context.Context, db Transaction) (*[]model.Worker, error) {
	rows, err := db.QueryContext(ctx, "SELECT name, ip, queue_name, last_seen FROM workers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workers := []model.Worker{}
	for rows.Next() {
		worker := model.Worker{}
		if err := rows.Scan(&worker.Name, &worker.Ip, &worker.QueueName, &worker.LastSeen); err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}

	return &workers, nil
}

func (S *SQLRepository) GetJob(ctx context.Context, uuid string) (job *model.Job, returnError error) {
	db, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	job, err = S.getJob(ctx, db, uuid)
	return job, err
}

func (S *SQLRepository) DeleteJob(ctx context.Context, uuid string) error {
	db, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	err = S.deleteJob(db, uuid)
	return err
}

func (S *SQLRepository) GetJobs(ctx context.Context) (jobs *[]model.Job, returnError error) {
	db, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	jobs, err = S.getJobs(ctx, db)
	return jobs, err
}

func (S *SQLRepository) GetTimeoutJobs(ctx context.Context, timeout time.Duration) (taskEvent []*model.TaskEvent, returnError error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	taskEvent, err = S.getTimeoutJobs(ctx, conn, timeout)
	if err != nil {
		return nil, err
	}
	return taskEvent, nil
}

func (S *SQLRepository) getJob(ctx context.Context, tx Transaction, uuid string) (*model.Job, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, source_path, destination_path FROM jobs WHERE id=$1", uuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	job := model.Job{}
	found := false
	if rows.Next() {
		if err := rows.Scan(&job.Id, &job.SourcePath, &job.DestinationPath); err != nil {
			return nil, err
		}
		found = true
	}
	if !found {
		return nil, fmt.Errorf("%w, %s", ErrElementNotFound, uuid)
	}

	taskEvents, err := S.getTaskEvents(ctx, tx, job.Id.String())
	if err != nil {
		return nil, err
	}
	job.Events = taskEvents
	lastUpdate, status, statusPhase, statusMessage, _ := S.getJobStatus(ctx, tx, job.Id.String())

	if lastUpdate != nil {
		job.LastUpdate = lastUpdate
	}
	job.Status = status
	job.StatusPhase = statusPhase
	job.StatusMessage = statusMessage

	return &job, nil
}

func (S *SQLRepository) deleteJob(tx Transaction, uuid string) error {
	sqlResult, err := tx.Exec("DELETE FROM jobs WHERE id=$1", uuid)
	log.Debugf("query result: +%v", sqlResult)
	if err != nil {
		return err
	}
	return nil
}

func (S *SQLRepository) getJobs(ctx context.Context, tx Transaction) (*[]model.Job, error) {
	query := fmt.Sprintf(`
    SELECT v.id, v.source_path, v.destination_path, vs.event_time, vs.status, vs.notification_type, vs.message
    FROM jobs v
    INNER JOIN job_status vs ON v.id = vs.job_id
`)
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []model.Job{}
	for rows.Next() {
		job := model.Job{}
		if err := rows.Scan(&job.Id, &job.SourcePath, &job.DestinationPath, &job.LastUpdate, &job.Status, &job.StatusPhase, &job.StatusMessage); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return &jobs, nil
}

func (S *SQLRepository) getTaskEvents(ctx context.Context, tx Transaction, uuid string) ([]*model.TaskEvent, error) {
	rows, err := tx.QueryContext(ctx, "SELECT job_id, job_event_id, worker_name, event_time, event_type, notification_type, status, message FROM job_events WHERE job_id=$1 order by event_time asc", uuid)
	if err != nil {
		log.Errorf("no job events founds by uuid: %s", uuid)
		return nil, err
	}
	defer rows.Close()
	var taskEvents []*model.TaskEvent
	for rows.Next() {
		event := model.TaskEvent{}
		if err := rows.Scan(&event.Id, &event.EventID, &event.WorkerName, &event.EventTime, &event.EventType, &event.NotificationType, &event.Status, &event.Message); err != nil {
			return nil, err
		}
		taskEvents = append(taskEvents, &event)
	}
	log.Debugf("task events: %+v", taskEvents)
	return taskEvents, nil
}

func (S *SQLRepository) getJobStatus(ctx context.Context, tx Transaction, uuid string) (*time.Time, string, model.NotificationType, string, error) {
	var last_update time.Time
	var status string
	var statusPhase model.NotificationType
	var message string

	rows, err := tx.QueryContext(ctx, "SELECT event_time, status, notification_type, message FROM job_status WHERE job_id=$1", uuid)
	if err != nil {
		return &last_update, status, statusPhase, message, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&last_update, &status, &statusPhase, &message); err != nil {
			return &last_update, status, statusPhase, message, err
		}
	}
	return &last_update, status, statusPhase, message, nil
}

func (S *SQLRepository) getJobByPath(ctx context.Context, tx Transaction, path string) (*model.Job, error) {
	rows, err := tx.QueryContext(ctx, "SELECT * FROM jobs WHERE source_path=$1", path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	job := model.Job{}

	found := false
	if rows.Next() {
		if err := rows.Scan(&job.Id, &job.SourcePath, &job.DestinationPath); err != nil {
			return nil, err
		}
		found = true
	}
	if !found {
		return nil, nil
	}

	taskEvents, err := S.getTaskEvents(ctx, tx, job.Id.String())
	if err != nil {
		return nil, err
	}
	job.Events = taskEvents
	return &job, nil
}

func (S *SQLRepository) GetJobByPath(ctx context.Context, path string) (*model.Job, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	return S.getJobByPath(ctx, conn, path)
}

func (S *SQLRepository) PingServerUpdate(ctx context.Context, name string, queueName string, ip string) (returnError error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "INSERT INTO workers (name, ip,queue_name,last_seen ) VALUES ($1,$2,$3,$4) ON CONFLICT (name) DO UPDATE SET ip = $2, queue_name=$3, last_seen=$4;", name, ip, queueName, time.Now())
	return err
}

func (S *SQLRepository) AddNewTaskEvent(ctx context.Context, event *model.TaskEvent) (returnError error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	return S.addNewTaskEvent(ctx, conn, event)
}

func (S *SQLRepository) addNewTaskEvent(ctx context.Context, tx Transaction, event *model.TaskEvent) error {
	rows, err := tx.QueryContext(ctx, "SELECT max(job_event_id) FROM job_events WHERE job_id=$1", event.Id.String())
	if err != nil {
		return err
	}

	var maxEventID sql.NullInt64
	if rows.Next() {
		if err := rows.Scan(&maxEventID); err != nil {
			rows.Close()
			return err
		}
	}
	jobEventID := -1
	if maxEventID.Valid {
		jobEventID = int(maxEventID.Int64)
	}
	if jobEventID+1 != event.EventID {
		rows.Close()
		return fmt.Errorf("EventID for %s not match,lastReceived %d, new %d", event.Id.String(), jobEventID, event.EventID)
	}

	rows.Close()
	_, err = tx.ExecContext(ctx, "INSERT INTO job_events (job_id, job_event_id,worker_name,event_time,event_type,notification_type,status,message)"+
		" VALUES ($1,$2,$3,$4,$5,$6,$7,$8)", event.Id.String(), event.EventID, event.WorkerName, time.Now(), event.EventType, event.NotificationType, event.Status, strings.TrimSpace(event.Message))
	return err
}
func (S *SQLRepository) AddJob(ctx context.Context, job *model.Job) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	return S.addJob(ctx, conn, job)
}

func (S *SQLRepository) addJob(ctx context.Context, tx Transaction, job *model.Job) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO jobs (id, source_path,destination_path)"+
		" VALUES ($1,$2,$3)", job.Id.String(), job.SourcePath, job.DestinationPath)
	return err
}

func (S *SQLRepository) getTimeoutJobs(ctx context.Context, tx Transaction, timeout time.Duration) ([]*model.TaskEvent, error) {
	timeoutDate := time.Now().Add(-timeout)

	rows, err := tx.QueryContext(ctx, "SELECT v.* FROM job_events v right join "+
		"(SELECT job_id,max(job_event_id) as job_event_id  FROM job_events WHERE notification_type='Job'  group by job_id) as m "+
		"on m.job_id=v.job_id and m.job_event_id=v.job_event_id WHERE status='started' and v.event_time < $1::timestamptz", timeoutDate)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var taskEvents []*model.TaskEvent
	for rows.Next() {
		event := model.TaskEvent{}
		if err := rows.Scan(&event.Id, &event.EventID, &event.WorkerName, &event.EventTime, &event.EventType, &event.NotificationType, &event.Status, &event.Message); err != nil {
			return nil, err
		}
		taskEvents = append(taskEvents, &event)
	}
	return taskEvents, nil
}

func (S *SQLRepository) WithTransaction(ctx context.Context, transactionFunc func(ctx context.Context, tx Repository) error) error {
	sqlTx, err := S.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return err
	}
	txRepository := *S
	txRepository.con = &SQLTransaction{
		tx: sqlTx,
	}

	err = transactionFunc(ctx, &txRepository)
	if err != nil {
		sqlTx.Rollback()
		return err
	} else {
		if err = sqlTx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (S *SQLRepository) GetDB() *sql.DB {
	return S.db
}

func (S *SQLRepository) EnqueueEncodeJob(ctx context.Context, task *model.TaskEncode) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		"INSERT INTO encode_queue (job_id, download_url, upload_url, checksum_url, event_id) VALUES ($1, $2, $3, $4, $5)",
		task.Id.String(), task.DownloadURL, task.UploadURL, task.ChecksumURL, task.EventID)
	return err
}

func (S *SQLRepository) DequeueEncodeJob(ctx context.Context, workerName string) (*model.TaskEncode, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	var task model.TaskEncode
	var jobID string
	err = conn.QueryRowContext(ctx, `
		UPDATE encode_queue 
		SET status = 'processing', locked_at = NOW(), locked_by = $1
		WHERE id = (
			SELECT id FROM encode_queue 
			WHERE status = 'pending'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING job_id, download_url, upload_url, checksum_url, event_id
	`, workerName).Scan(&jobID, &task.DownloadURL, &task.UploadURL, &task.ChecksumURL, &task.EventID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	task.Id, err = uuid.Parse(jobID)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (S *SQLRepository) EnqueuePGSJob(ctx context.Context, pgs *model.TaskPGS) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		"INSERT INTO pgs_queue (job_id, pgs_id, pgs_data, pgs_language, reply_to_queue) VALUES ($1, $2, $3, $4, $5)",
		pgs.Id.String(), pgs.PGSID, pgs.PGSdata, pgs.PGSLanguage, pgs.ReplyTo)
	return err
}

func (S *SQLRepository) DequeuePGSJob(ctx context.Context, workerName string) (*model.TaskPGS, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	var pgs model.TaskPGS
	var jobID string
	err = conn.QueryRowContext(ctx, `
		UPDATE pgs_queue 
		SET status = 'processing', locked_at = NOW(), locked_by = $1
		WHERE id = (
			SELECT id FROM pgs_queue 
			WHERE status = 'pending'
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING job_id, pgs_id, pgs_data, pgs_language, reply_to_queue
	`, workerName).Scan(&jobID, &pgs.PGSID, &pgs.PGSdata, &pgs.PGSLanguage, &pgs.ReplyTo)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pgs.Id, err = uuid.Parse(jobID)
	if err != nil {
		return nil, err
	}
	return &pgs, nil
}

func (S *SQLRepository) EnqueuePGSResponse(ctx context.Context, resp *model.TaskPGSResponse) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		"INSERT INTO pgs_responses (job_id, pgs_id, srt_data, error, reply_to_queue) VALUES ($1, $2, $3, $4, $5)",
		resp.Id.String(), resp.PGSID, resp.Srt, resp.Err, resp.Queue)
	return err
}

func (S *SQLRepository) DequeuePGSResponse(ctx context.Context, replyToQueue string) (*model.TaskPGSResponse, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	var resp model.TaskPGSResponse
	var jobID string
	err = conn.QueryRowContext(ctx, `
		UPDATE pgs_responses 
		SET consumed = true, consumed_at = NOW()
		WHERE id = (
			SELECT id FROM pgs_responses 
			WHERE reply_to_queue = $1 AND consumed = false
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING job_id, pgs_id, srt_data, error
	`, replyToQueue).Scan(&jobID, &resp.PGSID, &resp.Srt, &resp.Err)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	resp.Id, err = uuid.Parse(jobID)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (S *SQLRepository) EnqueueTaskEvent(ctx context.Context, event *model.TaskEvent) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		`INSERT INTO task_event_queue (job_id, event_id, event_type, worker_name, worker_queue, event_time, ip, notification_type, status, message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		event.Id.String(), event.EventID, event.EventType, event.WorkerName, event.WorkerQueue, event.EventTime, event.IP, event.NotificationType, event.Status, event.Message)
	return err
}

func (S *SQLRepository) DequeueTaskEvents(ctx context.Context, limit int) ([]*model.TaskEvent, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := conn.QueryContext(ctx, `
		DELETE FROM task_event_queue
		WHERE id IN (
			SELECT id FROM task_event_queue
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING job_id, event_id, event_type, worker_name, worker_queue, event_time, ip, notification_type, status, message
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.TaskEvent
	for rows.Next() {
		var event model.TaskEvent
		var jobID string
		if err := rows.Scan(&jobID, &event.EventID, &event.EventType, &event.WorkerName, &event.WorkerQueue, &event.EventTime, &event.IP, &event.NotificationType, &event.Status, &event.Message); err != nil {
			return nil, err
		}
		event.Id, err = uuid.Parse(jobID)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}
	return events, nil
}

func (S *SQLRepository) EnqueueJobAction(ctx context.Context, jobID string, workerName string, action model.JobAction) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		"INSERT INTO job_actions (job_id, worker_name, action) VALUES ($1, $2, $3)",
		jobID, workerName, action)
	return err
}

func (S *SQLRepository) DequeueJobActions(ctx context.Context, workerName string) ([]*model.JobEvent, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := conn.QueryContext(ctx, `
		UPDATE job_actions 
		SET consumed = true, consumed_at = NOW()
		WHERE id IN (
			SELECT id FROM job_actions 
			WHERE worker_name = $1 AND consumed = false
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
		)
		RETURNING job_id, action
	`, workerName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []*model.JobEvent
	for rows.Next() {
		var action model.JobEvent
		var jobID string
		if err := rows.Scan(&jobID, &action.Action); err != nil {
			return nil, err
		}
		action.Id, err = uuid.Parse(jobID)
		if err != nil {
			return nil, err
		}
		actions = append(actions, &action)
	}
	return actions, nil
}

func (S *SQLRepository) AddFileProcessing(ctx context.Context, fp *model.FileProcessing) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}

	var jobID interface{}
	if fp.JobId != nil {
		jobID = fp.JobId.String()
	}

	_, err = conn.ExecContext(ctx, `
		INSERT INTO file_processing (path, detected_at, source, status, message, job_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (path) DO UPDATE SET
			detected_at = EXCLUDED.detected_at,
			source = EXCLUDED.source,
			status = EXCLUDED.status,
			message = EXCLUDED.message,
			job_id = EXCLUDED.job_id
	`, fp.Path, fp.DetectedAt, fp.Source, fp.Status, fp.Message, jobID)
	return err
}

func (S *SQLRepository) GetFileProcessingByPath(ctx context.Context, path string) (*model.FileProcessing, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	var fp model.FileProcessing
	var jobID sql.NullString

	err = conn.QueryRowContext(ctx, `
		SELECT id, path, detected_at, source, status, message, job_id, created_at
		FROM file_processing WHERE path = $1
	`, path).Scan(&fp.Id, &fp.Path, &fp.DetectedAt, &fp.Source, &fp.Status, &fp.Message, &jobID, &fp.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if jobID.Valid {
		id, err := uuid.Parse(jobID.String)
		if err == nil {
			fp.JobId = &id
		}
	}

	return &fp, nil
}

func (S *SQLRepository) GetRecentFileProcessings(ctx context.Context, limit int, source model.FileProcessingSource) ([]*model.FileProcessing, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	var query string
	var args []interface{}

	if source == "" {
		query = `
			SELECT id, path, detected_at, source, status, message, job_id, created_at
			FROM file_processing
			ORDER BY detected_at DESC
			LIMIT $1
		`
		args = []interface{}{limit}
	} else {
		query = `
			SELECT id, path, detected_at, source, status, message, job_id, created_at
			FROM file_processing
			WHERE source = $1
			ORDER BY detected_at DESC
			LIMIT $2
		`
		args = []interface{}{source, limit}
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var processings []*model.FileProcessing
	for rows.Next() {
		var fp model.FileProcessing
		var jobID sql.NullString
		if err := rows.Scan(&fp.Id, &fp.Path, &fp.DetectedAt, &fp.Source, &fp.Status, &fp.Message, &jobID, &fp.CreatedAt); err != nil {
			return nil, err
		}
		if jobID.Valid {
			id, err := uuid.Parse(jobID.String)
			if err == nil {
				fp.JobId = &id
			}
		}
		processings = append(processings, &fp)
	}

	return processings, nil
}

func (S *SQLRepository) CreateScan(ctx context.Context, scan *model.LibraryScan) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		`INSERT INTO library_scans (id, started_at, completed_at, status, files_found, files_queued, files_skipped_size, files_skipped_codec, files_skipped_exists, error_message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		scan.Id, scan.StartedAt, scan.CompletedAt, scan.Status, scan.FilesFound, scan.FilesQueued,
		scan.FilesSkippedSize, scan.FilesSkippedCodec, scan.FilesSkippedExist, scan.ErrorMessage)
	return err
}

func (S *SQLRepository) UpdateScan(ctx context.Context, scan *model.LibraryScan) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		`UPDATE library_scans SET completed_at=$1, status=$2, files_found=$3, files_queued=$4, 
		 files_skipped_size=$5, files_skipped_codec=$6, files_skipped_exists=$7, error_message=$8
		 WHERE id=$9`,
		scan.CompletedAt, scan.Status, scan.FilesFound, scan.FilesQueued,
		scan.FilesSkippedSize, scan.FilesSkippedCodec, scan.FilesSkippedExist, scan.ErrorMessage, scan.Id)
	return err
}

func (S *SQLRepository) GetScan(ctx context.Context, id string) (*model.LibraryScan, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx,
		`SELECT id, started_at, completed_at, status, files_found, files_queued, 
		 files_skipped_size, files_skipped_codec, files_skipped_exists, error_message
		 FROM library_scans WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		scan := &model.LibraryScan{}
		if err := rows.Scan(&scan.Id, &scan.StartedAt, &scan.CompletedAt, &scan.Status,
			&scan.FilesFound, &scan.FilesQueued, &scan.FilesSkippedSize, &scan.FilesSkippedCodec,
			&scan.FilesSkippedExist, &scan.ErrorMessage); err != nil {
			return nil, err
		}
		return scan, nil
	}
	return nil, fmt.Errorf("%w: scan %s", ErrElementNotFound, id)
}

func (S *SQLRepository) GetLatestScan(ctx context.Context) (*model.LibraryScan, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx,
		`SELECT id, started_at, completed_at, status, files_found, files_queued, 
		 files_skipped_size, files_skipped_codec, files_skipped_exists, error_message
		 FROM library_scans ORDER BY started_at DESC LIMIT 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		scan := &model.LibraryScan{}
		if err := rows.Scan(&scan.Id, &scan.StartedAt, &scan.CompletedAt, &scan.Status,
			&scan.FilesFound, &scan.FilesQueued, &scan.FilesSkippedSize, &scan.FilesSkippedCodec,
			&scan.FilesSkippedExist, &scan.ErrorMessage); err != nil {
			return nil, err
		}
		return scan, nil
	}
	return nil, nil
}

func (S *SQLRepository) GetScanHistory(ctx context.Context, limit int) ([]*model.LibraryScan, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx,
		`SELECT id, started_at, completed_at, status, files_found, files_queued, 
		 files_skipped_size, files_skipped_codec, files_skipped_exists, error_message
		 FROM library_scans ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []*model.LibraryScan
	for rows.Next() {
		scan := &model.LibraryScan{}
		if err := rows.Scan(&scan.Id, &scan.StartedAt, &scan.CompletedAt, &scan.Status,
			&scan.FilesFound, &scan.FilesQueued, &scan.FilesSkippedSize, &scan.FilesSkippedCodec,
			&scan.FilesSkippedExist, &scan.ErrorMessage); err != nil {
			return nil, err
		}
		scans = append(scans, scan)
	}
	return scans, nil
}

func (S *SQLRepository) UpsertScannedFile(ctx context.Context, file *model.ScannedFile) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx,
		`INSERT INTO scanned_files (id, file_path, file_size, codec, last_scanned_at, queued, scan_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (file_path) DO UPDATE SET 
		 file_size=$3, codec=$4, last_scanned_at=$5, queued=$6, scan_id=$7`,
		file.Id, file.FilePath, file.FileSize, file.Codec, file.LastScannedAt, file.Queued, file.ScanId)
	return err
}

func (S *SQLRepository) GetScannedFile(ctx context.Context, path string) (*model.ScannedFile, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx,
		`SELECT id, file_path, file_size, codec, last_scanned_at, queued, scan_id
		 FROM scanned_files WHERE file_path=$1`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		file := &model.ScannedFile{}
		if err := rows.Scan(&file.Id, &file.FilePath, &file.FileSize, &file.Codec,
			&file.LastScannedAt, &file.Queued, &file.ScanId); err != nil {
			return nil, err
		}
		return file, nil
	}
	return nil, fmt.Errorf("%w: file %s", ErrElementNotFound, path)
}

func (S *SQLRepository) GetScannedFilesByScan(ctx context.Context, scanID string) ([]*model.ScannedFile, error) {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx,
		`SELECT id, file_path, file_size, codec, last_scanned_at, queued, scan_id
		 FROM scanned_files WHERE scan_id=$1 ORDER BY last_scanned_at DESC`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*model.ScannedFile
	for rows.Next() {
		file := &model.ScannedFile{}
		if err := rows.Scan(&file.Id, &file.FilePath, &file.FileSize, &file.Codec,
			&file.LastScannedAt, &file.Queued, &file.ScanId); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

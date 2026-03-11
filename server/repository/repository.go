package repository

import (
	"context"
	"database/sql"
	"fmt"
	"gearr/model"
	"strings"
	"time"

	_ "embed"

	_ "github.com/lib/pq"
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
}

type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
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
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode)
	db, err := sql.Open(config.Driver, connectionString)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(5)
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
		rows.Scan(&worker.Name, &worker.Ip, &worker.QueueName, &worker.LastSeen)
		found = true
	}
	if !found {
		return nil, fmt.Errorf("%w, %s", ErrElementNotFound, name)
	}
	return &worker, err
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
		rows.Scan(&worker.Name, &worker.Ip, &worker.QueueName, &worker.LastSeen)
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
	job := model.Job{}
	found := false
	if rows.Next() {
		rows.Scan(&job.Id, &job.SourcePath, &job.DestinationPath)
		found = true
	}
	rows.Close()
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
		rows.Scan(&job.Id, &job.SourcePath, &job.DestinationPath, &job.LastUpdate, &job.Status, &job.StatusPhase, &job.StatusMessage)
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
		rows.Scan(&event.Id, &event.EventID, &event.WorkerName, &event.EventTime, &event.EventType, &event.NotificationType, &event.Status, &event.Message)
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
		rows.Scan(&last_update, &status, &statusPhase, &message)
	}
	return &last_update, status, statusPhase, message, nil
}

func (S *SQLRepository) getJobByPath(ctx context.Context, tx Transaction, path string) (*model.Job, error) {
	log.Debugf("get job by path: %s", path)
	rows, err := tx.QueryContext(ctx, "SELECT * FROM jobs WHERE source_path=$1", path)
	if err != nil {
		log.Errorf("no job founds by path: %s", path)
		return nil, err
	}

	log.Debugf("rows: %+v", rows)

	job := model.Job{}

	found := false
	if rows.Next() {
		rows.Scan(&job.Id, &job.SourcePath, &job.DestinationPath)
		found = true
	}
	log.Debugf("job: %+v", job)
	rows.Close()
	if !found {
		return nil, nil
	}

	taskEvents, err := S.getTaskEvents(ctx, tx, job.Id.String())
	log.Debugf("taskEvents: %+v", taskEvents)
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

	jobEventID := -1
	if rows.Next() {
		rows.Scan(&jobEventID)
	}
	rows.Close()
	if jobEventID+1 != event.EventID {
		return fmt.Errorf("EventID for %s not match,lastReceived %d, new %d", event.Id.String(), jobEventID, event.EventID)
	}

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

	//2020-05-17 20:50:41.428531 +00:00
	if err != nil {
		return nil, err
	}
	var taskEvents []*model.TaskEvent
	for rows.Next() {
		event := model.TaskEvent{}
		rows.Scan(&event.Id, &event.EventID, &event.WorkerName, &event.EventTime, &event.EventType, &event.NotificationType, &event.Status, &event.Message)
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

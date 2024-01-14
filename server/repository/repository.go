package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"transcoder/model"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var (
	ElementNotFound = fmt.Errorf("element not found")
)

type Repository interface {
	getConnection(ctx context.Context) (Transaction, error)
	Initialize(ctx context.Context) error
	ProcessEvent(ctx context.Context, event *model.TaskEvent) error
	PingServerUpdate(ctx context.Context, name string, ip string, queueName string) error
	GetTimeoutJobs(ctx context.Context, timeout time.Duration) ([]*model.TaskEvent, error)
	GetJob(ctx context.Context, uuid string) (*model.Video, error)
	GetJobs(ctx context.Context) (*[]model.Video, error)
	GetJobByPath(ctx context.Context, path string) (*model.Video, error)
	AddNewTaskEvent(ctx context.Context, event *model.TaskEvent) error
	AddVideo(ctx context.Context, video *model.Video) error
	WithTransaction(ctx context.Context, transactionFunc func(ctx context.Context, tx Repository) error) error
	GetWorker(ctx context.Context, name string) (*model.Worker, error)
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
	db     *sql.DB
	con    Transaction
	assets http.FileSystem
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

func NewSQLRepository(config SQLServerConfig, assets http.FileSystem) (*SQLRepository, error) {
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
		db:     db,
		assets: assets,
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

func (S *SQLRepository) prepareDatabase(ctx context.Context) (returnError error) {
	schemeName, err := S.assets.Open("/database/database.sql")
	if err != nil {
		return err
	}
	filebytes, err := ioutil.ReadAll(schemeName)
	if err != nil {
		return err
	}
	databaseScript := string(filebytes)
	err = S.WithTransaction(ctx, func(ctx context.Context, tx Repository) error {
		con, err := tx.getConnection(ctx)
		if err != nil {
			return err
		}
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
	worker, err = S.getWorker(ctx, db, name)
	return worker, err
}
func (S *SQLRepository) getWorker(ctx context.Context, db Transaction, name string) (*model.Worker, error) {
	rows, err := db.QueryContext(ctx, "select * from workers where name=$1", name)
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
		return nil, fmt.Errorf("%w, %s", ElementNotFound, name)
	}
	return &worker, err
}

func (S *SQLRepository) GetJob(ctx context.Context, uuid string) (video *model.Video, returnError error) {
	db, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	video, err = S.getJob(ctx, db, uuid)
	return video, err
}

func (S *SQLRepository) GetJobs(ctx context.Context) (videos *[]model.Video, returnError error) {
	db, err := S.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	videos, err = S.getJobs(ctx, db)
	return videos, err
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

func (S *SQLRepository) getJob(ctx context.Context, tx Transaction, uuid string) (*model.Video, error) {
	rows, err := tx.QueryContext(ctx, "select * from videos where id=$1", uuid)
	if err != nil {
		return nil, err
	}
	video := model.Video{}
	found := false
	if rows.Next() {
		rows.Scan(&video.Id, &video.SourcePath, &video.DestinationPath)
		found = true
	}
	rows.Close()
	if !found {
		return nil, fmt.Errorf("%w, %s", ElementNotFound, uuid)
	}

	taskEvents, err := S.getTaskEvents(ctx, tx, video.Id.String())
	if err != nil {
		return nil, err
	}
	video.Events = taskEvents
	return &video, nil
}

func (S *SQLRepository) getJobs(ctx context.Context, tx Transaction) (*[]model.Video, error) {
	rows, err := tx.QueryContext(ctx, "select id from videos")
	if err != nil {
		return nil, err
	}
	videos := []model.Video{}
	if rows.Next() {
		video := model.Video{}
		rows.Scan(&video.Id)
		videos = append(videos, video)
	}
	rows.Close()
	return &videos, nil
}

func (S *SQLRepository) getTaskEvents(ctx context.Context, tx Transaction, uuid string) ([]*model.TaskEvent, error) {
	rows, err := tx.QueryContext(ctx, "select * from video_events where video_id=$1 order by event_time asc", uuid)
	if err != nil {
		log.Errorf("no video events founds by uuid: %s", uuid)
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
func (S *SQLRepository) getJobByPath(ctx context.Context, tx Transaction, path string) (*model.Video, error) {
	log.Debugf("get video by path: %s", path)
	rows, err := tx.QueryContext(ctx, "select * from videos where source_path=$1", path)
	if err != nil {
		log.Errorf("no video founds by path: %s", path)
		return nil, err
	}

	log.Debugf("rows: %+v", rows)

	video := model.Video{}

	found := false
	if rows.Next() {
		rows.Scan(&video.Id, &video.SourcePath, &video.DestinationPath)
		found = true
	}
	log.Debugf("video: %+v", video)
	rows.Close()
	if !found {
		return nil, nil
	}

	taskEvents, err := S.getTaskEvents(ctx, tx, video.Id.String())
	log.Debugf("taskEvents: %+v", taskEvents)
	if err != nil {
		return nil, err
	}
	video.Events = taskEvents
	return &video, nil
}

func (S *SQLRepository) GetJobByPath(ctx context.Context, path string) (video *model.Video, returnError error) {
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
	rows, err := tx.QueryContext(ctx, "select max(video_event_id) from video_events where video_id=$1", event.Id.String())
	if err != nil {
		return err
	}

	videoEventID := -1
	if rows.Next() {
		rows.Scan(&videoEventID)
	}
	rows.Close()
	if videoEventID+1 != event.EventID {
		return fmt.Errorf("EventID for %s not match,lastReceived %d, new %d", event.Id.String(), videoEventID, event.EventID)
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO video_events (video_id, video_event_id,worker_name,event_time,event_type,notification_type,status,message)"+
		" VALUES ($1,$2,$3,$4,$5,$6,$7,$8)", event.Id.String(), event.EventID, event.WorkerName, time.Now(), event.EventType, event.NotificationType, event.Status, event.Message)
	return err
}
func (S *SQLRepository) AddVideo(ctx context.Context, video *model.Video) error {
	conn, err := S.getConnection(ctx)
	if err != nil {
		return err
	}
	return S.addVideo(ctx, conn, video)
}

func (S *SQLRepository) addVideo(ctx context.Context, tx Transaction, video *model.Video) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO videos (id, source_path,destination_path)"+
		" VALUES ($1,$2,$3)", video.Id.String(), video.SourcePath, video.DestinationPath)
	return err
}

func (S *SQLRepository) getTimeoutJobs(ctx context.Context, tx Transaction, timeout time.Duration) ([]*model.TaskEvent, error) {
	timeoutDate := time.Now().Add(-timeout)
	timeoutDate.Format(time.RFC3339)

	rows, err := tx.QueryContext(ctx, "select v.* from video_events v right join "+
		"(select video_id,max(video_event_id) as video_event_id  from video_events where notification_type='Job'  group by video_id) as m "+
		"on m.video_id=v.video_id and m.video_event_id=v.video_event_id where status='started' and v.event_time < $1::timestamptz", timeoutDate)

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

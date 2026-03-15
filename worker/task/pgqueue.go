package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gearr/helper"
	"gearr/helper/concurrent"
	"gearr/model"
	"gearr/server/repository"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
)

type PostgresClient struct {
	db                *sql.DB
	repo              *repository.SQLRepository
	workerConfig      Config
	workerUniqueQueue string
	PGSWorker         []*JobWorker
	EncodeWorker      *JobWorker
	printer           *ConsoleWorkerPrinter
	pollInterval      time.Duration
	pgsJobControls    *concurrent.Map
}

func NewBrokerClientPostgres(dbConfig repository.SQLServerConfig, workerConfig Config, printer *ConsoleWorkerPrinter) (*PostgresClient, error) {
	repo, err := repository.NewSQLRepository(dbConfig)
	if err != nil {
		return nil, err
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	uniqueID := fmt.Sprintf("%s-%d", workerConfig.Name, rnd.Intn(5000000))

	pgsJobControls := &concurrent.Map{}
	pgsJobControls.Set("_init", nil)

	return &PostgresClient{
		repo:              repo,
		db:                repo.GetDB(),
		workerConfig:      workerConfig,
		workerUniqueQueue: uniqueID,
		printer:           printer,
		pollInterval:      time.Second,
		pgsJobControls:    pgsJobControls,
	}, nil
}

func (p *PostgresClient) RegisterPGSWorker(worker *PGSWorker) {
	worker.Manager = p
	p.PGSWorker = append(p.PGSWorker, &JobWorker{
		active:    false,
		pgsWorker: worker,
		pgs:       concurrent.Slice{},
	})
}

func (p *PostgresClient) RegisterEncodeWorker(worker *EncodeWorker) {
	worker.Manager = p
	p.EncodeWorker = &JobWorker{
		active:       false,
		encodeWorker: worker,
		pgs:          concurrent.Slice{},
	}
}

func (p *PostgresClient) Run(wg *sync.WaitGroup, ctx context.Context) {
	log.Info("starting postgres broker client")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("stopping postgres broker client")
		wg.Done()
	}()

	go p.eventProcessor(ctx)
}

func (p *PostgresClient) EventNotification(event model.TaskEvent) error {
	_, err := p.db.ExecContext(context.Background(),
		`INSERT INTO task_event_queue (job_id, event_id, event_type, worker_name, worker_queue, event_time, ip, notification_type, status, message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		event.Id.String(), event.EventID, event.EventType, event.WorkerName, event.WorkerQueue, event.EventTime, event.IP, event.NotificationType, event.Status, event.Message)
	if err != nil {
		return err
	}

	log.Debugf("[job %s] %s has been %s", event.Id.String(), event.NotificationType, event.Status)
	return nil
}

func (p *PostgresClient) RequestPGSJob(pgsJob model.TaskPGS) <-chan *model.TaskPGSResponse {
	pgsJobControl := NewPGSJobControl(pgsJob)
	pgsJob.ReplyTo = p.workerUniqueQueue

	_, err := p.db.ExecContext(context.Background(),
		"INSERT INTO pgs_queue (job_id, pgs_id, pgs_data, pgs_language, reply_to_queue) VALUES ($1, $2, $3, $4, $5)",
		pgsJob.Id.String(), pgsJob.PGSID, pgsJob.PGSdata, pgsJob.PGSLanguage, pgsJob.ReplyTo)

	if err != nil {
		log.Errorf("failed to publish PGS job: %v", err)
		pgsJobControl.response <- &model.TaskPGSResponse{
			Id:    pgsJob.Id,
			PGSID: pgsJob.PGSID,
			Err:   err.Error(),
		}
		close(pgsJobControl.response)
		return pgsJobControl.response
	}

	log.Debugf("published PGS job %s", pgsJobControl.task.Id)
	p.EncodeWorker.pgs.Append(pgsJobControl)
	p.pgsJobControls.Set(fmt.Sprintf("%d", pgsJob.PGSID), pgsJobControl)
	return pgsJobControl.response
}

func (p *PostgresClient) ResponsePGSJob(pgsResponse model.TaskPGSResponse) error {
	_, err := p.db.ExecContext(context.Background(),
		"INSERT INTO pgs_responses (job_id, pgs_id, srt_data, error, reply_to_queue) VALUES ($1, $2, $3, $4, $5)",
		pgsResponse.Id.String(), pgsResponse.PGSID, pgsResponse.Srt, pgsResponse.Err, pgsResponse.Queue)
	return err
}

func (p *PostgresClient) eventProcessor(ctx context.Context) {
	if p.workerConfig.Jobs.IsAccepted(model.PGSToSrtJobType) {
		go p.pgsQueueProcessor(ctx)
	}
	if p.workerConfig.Jobs.IsAccepted(model.EncodeJobType) {
		go p.encodeQueueProcessor(ctx)
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 30):
			pingEvent := model.TaskEvent{
				EventType:   model.PingEvent,
				WorkerName:  p.workerConfig.Name,
				WorkerQueue: p.workerUniqueQueue,
				EventTime:   time.Now(),
				IP:          helper.GetPublicIP(),
			}
			p.EventNotification(pingEvent)
		case <-ticker.C:
			p.checkPGSResponses()
			p.checkJobActions(ctx)
		}
	}
}

func (p *PostgresClient) checkPGSResponses() {
	rows, err := p.db.QueryContext(context.Background(), `
		UPDATE pgs_responses 
		SET consumed = true, consumed_at = NOW()
		WHERE id IN (
			SELECT id FROM pgs_responses 
			WHERE reply_to_queue = $1 AND consumed = false
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
		)
		RETURNING job_id, pgs_id, srt_data, error
	`, p.workerUniqueQueue)

	if err != nil {
		log.Errorf("failed to check PGS responses: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var resp model.TaskPGSResponse
		var jobID string
		if err := rows.Scan(&jobID, &resp.PGSID, &resp.Srt, &resp.Err); err != nil {
			log.Errorf("failed to scan PGS response: %v", err)
			continue
		}

		if val, ok := p.pgsJobControls.Get(fmt.Sprintf("%d", resp.PGSID)); ok {
			pgsJobControl := val.(*TaskPGSJobControl)
			resp.Id = pgsJobControl.task.Id
			pgsJobControl.response <- &resp
			close(pgsJobControl.response)
			p.EncodeWorker.pgs.Delete(pgsJobControl)
		}
	}
}

func (p *PostgresClient) checkJobActions(ctx context.Context) {
	rows, err := p.db.QueryContext(ctx, `
		UPDATE job_actions 
		SET consumed = true, consumed_at = NOW()
		WHERE id IN (
			SELECT id FROM job_actions 
			WHERE worker_name = $1 AND consumed = false
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
		)
		RETURNING job_id, action
	`, p.workerUniqueQueue)

	if err != nil {
		log.Errorf("failed to check job actions: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var action model.JobAction
		var jobID string
		if err := rows.Scan(&jobID, &action); err != nil {
			log.Errorf("failed to scan job action: %v", err)
			continue
		}
		log.Infof("received job action %s for job %s", action, jobID)
	}
}

func (p *PostgresClient) pgsQueueProcessor(ctx context.Context) {
	log.Info("starting PGS queue processor")
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, worker := range p.PGSWorker {
				if !worker.active && worker.pgsWorker.AcceptJobs() {
					var pgsJob model.TaskPGS
					var jobID string
					err := p.db.QueryRowContext(ctx, `
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
					`, p.workerUniqueQueue).Scan(&jobID, &pgsJob.PGSID, &pgsJob.PGSdata, &pgsJob.PGSLanguage, &pgsJob.ReplyTo)

					if err == sql.ErrNoRows {
						continue
					}
					if err != nil {
						log.Errorf("failed to dequeue PGS job: %v", err)
						continue
					}

					pgsJob.Id, err = uuid.Parse(jobID)
					if err != nil {
						log.Errorf("failed to parse job ID: %v", err)
						continue
					}

					p.printer.Log("[%s] Job Assigned to %s", model.PGSToSrtJobType, worker.pgsWorker.GetID())
					pgsJobData, err := json.Marshal(pgsJob)
					if err != nil {
						log.Errorf("failed to marshal PGS job: %v", err)
						continue
					}
					if err := worker.pgsWorker.Prepare(pgsJobData, p); err != nil {
						worker.pgsWorker.Clean()
						p.printer.Error("[%s] Error preparing job execution on %s", model.PGSToSrtJobType, worker.pgsWorker.GetID())
						continue
					}
					worker.jobID = worker.pgsWorker.GetTaskID()
					worker.active = true
					go p.controlPGSJobExecution(worker)
				}
			}
		}
	}
}

func (p *PostgresClient) encodeQueueProcessor(ctx context.Context) {
	log.Info("starting encode queue processor")
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	log.Debug("start encode worker manager")
	p.EncodeWorker.encodeWorker.Manager = p

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if p.EncodeWorker.encodeWorker.AcceptJobs() {
				var task model.TaskEncode
				var jobID string
				err := p.db.QueryRowContext(ctx, `
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
				`, p.workerUniqueQueue).Scan(&jobID, &task.DownloadURL, &task.UploadURL, &task.ChecksumURL, &task.EventID)

				if err == sql.ErrNoRows {
					continue
				}
				if err != nil {
					log.Errorf("failed to dequeue encode job: %v", err)
					continue
				}

				task.Id, err = uuid.Parse(jobID)
				if err != nil {
					log.Errorf("failed to parse job ID: %v", err)
					continue
				}

				taskData, err := json.Marshal(task)
				if err != nil {
					log.Errorf("failed to marshal task: %v", err)
					continue
				}

				if err := p.EncodeWorker.encodeWorker.Execute(taskData); err != nil {
					log.Errorf("[%s] Error Preparing Job Execution: %v", model.EncodeJobType, err)
					continue
				}
				log.Debug("execute a new encoder job")
			}
		}
	}
}

func (p *PostgresClient) controlPGSJobExecution(jobWorker *JobWorker) {
	defer func() {
		if err := jobWorker.pgsWorker.Clean(); err != nil {
			log.Errorf("error cleaning working path for worker %s: %v", jobWorker.pgsWorker.GetID(), err)
		}
		jobWorker.active = false
	}()
	jobWorker.pgsWorker.Execute()
}

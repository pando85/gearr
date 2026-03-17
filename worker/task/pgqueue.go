package task

import (
	"context"
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
	log "github.com/sirupsen/logrus"
)

type JobWorker struct {
	jobID        uuid.UUID
	active       bool
	pgs          concurrent.Slice
	pgsWorker    *PGSWorker
	encodeWorker *EncodeWorker
}

type TaskPGSJobControl struct {
	task     model.TaskPGS
	response chan *model.TaskPGSResponse
}

func NewPGSJobControl(task model.TaskPGS) *TaskPGSJobControl {
	return &TaskPGSJobControl{
		task:     task,
		response: make(chan *model.TaskPGSResponse, 1),
	}
}

type PostgresClient struct {
	repo              repository.Repository
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

	pgsJobControls := concurrent.NewMap()
	pgsJobControls.Set("_init", nil)

	return &PostgresClient{
		repo:              repo,
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
	log.Info("starting broker client")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("stopping broker client")
		wg.Done()
	}()

	go p.eventProcessor(ctx)
}

func (p *PostgresClient) EventNotification(event model.TaskEvent) error {
	err := p.repo.EnqueueTaskEvent(context.Background(), &event)
	if err != nil {
		return err
	}

	log.Debugf("[job %s] %s has been %s", event.Id.String(), event.NotificationType, event.Status)
	return nil
}

func (p *PostgresClient) RequestPGSJob(pgsJob model.TaskPGS) <-chan *model.TaskPGSResponse {
	pgsJobControl := NewPGSJobControl(pgsJob)
	pgsJob.ReplyTo = p.workerUniqueQueue

	err := p.repo.EnqueuePGSJob(context.Background(), &pgsJob)
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
	return p.repo.EnqueuePGSResponse(context.Background(), &pgsResponse)
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

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			publicIP, err := helper.GetPublicIP()
			if err != nil {
				log.Warnf("failed to get public IP: %v", err)
			}
			pingEvent := model.TaskEvent{
				EventType:   model.PingEvent,
				WorkerName:  p.workerConfig.Name,
				WorkerQueue: p.workerUniqueQueue,
				EventTime:   time.Now(),
				IP:          publicIP,
			}
			p.EventNotification(pingEvent)
		case <-ticker.C:
			p.checkPGSResponses()
			p.checkJobActions(ctx)
		}
	}
}

func (p *PostgresClient) checkPGSResponses() {
	resp, err := p.repo.DequeuePGSResponse(context.Background(), p.workerUniqueQueue)
	if err != nil {
		log.Errorf("failed to check PGS responses: %v", err)
		return
	}
	if resp == nil {
		return
	}

	if val, ok := p.pgsJobControls.Get(fmt.Sprintf("%d", resp.PGSID)); ok {
		pgsJobControl := val.(*TaskPGSJobControl)
		resp.Id = pgsJobControl.task.Id
		pgsJobControl.response <- resp
		close(pgsJobControl.response)
		p.EncodeWorker.pgs.Delete(pgsJobControl)
	}
}

func (p *PostgresClient) checkJobActions(ctx context.Context) {
	actions, err := p.repo.DequeueJobActions(ctx, p.workerUniqueQueue)
	if err != nil {
		log.Errorf("failed to check job actions: %v", err)
		return
	}
	for _, action := range actions {
		log.Infof("received job action %s for job %s", action.Action, action.Id.String())
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
					pgsJob, err := p.repo.DequeuePGSJob(ctx, p.workerUniqueQueue)
					if err != nil {
						log.Errorf("failed to dequeue PGS job: %v", err)
						continue
					}
					if pgsJob == nil {
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
				task, err := p.repo.DequeueEncodeJob(ctx, p.workerUniqueQueue)
				if err != nil {
					log.Errorf("failed to dequeue encode job: %v", err)
					continue
				}
				if task == nil {
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

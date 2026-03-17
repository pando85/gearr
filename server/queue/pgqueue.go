package queue

import (
	"context"
	"gearr/helper"
	"gearr/internal/constants"
	"gearr/model"
	"gearr/server/repository"
	"sync"
	"time"
)

type PostgresBrokerServer struct {
	repo               repository.Repository
	newTask            chan *model.ControlEvent
	newWorkerEvent     chan *model.JobEventQueue
	taskEventConsumers []chan *model.TaskEvent
	pollInterval       time.Duration
}

func NewBrokerServerPostgres(repo repository.Repository) (*PostgresBrokerServer, error) {
	return &PostgresBrokerServer{
		repo:           repo,
		newTask:        make(chan *model.ControlEvent, constants.ChannelBufferSize),
		newWorkerEvent: make(chan *model.JobEventQueue, constants.ChannelBufferSize),
		pollInterval:   time.Second,
	}, nil
}

func (p *PostgresBrokerServer) Run(wg *sync.WaitGroup, ctx context.Context) {
	helper.Info("starting postgres broker")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		helper.Info("stopping postgres broker")
		wg.Done()
	}()
	go p.taskPublisher(ctx)
	go p.taskEventProcessor(ctx)
}

func (p *PostgresBrokerServer) PublishJobRequest(taskRequest *model.TaskEncode) error {
	controlChan := make(chan interface{})
	p.newTask <- &model.ControlEvent{
		Event:       taskRequest,
		ControlChan: controlChan,
	}
	rtn := <-controlChan
	if rtn == nil {
		return nil
	}
	err := (rtn).(error)
	return err
}

func (p *PostgresBrokerServer) ReceiveJobEvent() <-chan *model.TaskEvent {
	tc := make(chan *model.TaskEvent, constants.ChannelBufferSize)
	p.taskEventConsumers = append(p.taskEventConsumers, tc)
	return tc
}

func (p *PostgresBrokerServer) PublishJobEvent(jobEvent *model.JobEvent, workerQueue string) {
	jobEventQueue := &model.JobEventQueue{
		Queue:    workerQueue,
		JobEvent: jobEvent,
	}
	p.newWorkerEvent <- jobEventQueue
}

func (p *PostgresBrokerServer) taskPublisher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case workerEvent := <-p.newWorkerEvent:
			err := p.repo.EnqueueJobAction(ctx, workerEvent.JobEvent.Id.String(), workerEvent.Queue, workerEvent.JobEvent.Action)
			if err != nil {
				helper.Errorf("failed to enqueue job action: %v", err)
			} else {
				helper.Infof("sending %s action for job %s", workerEvent.JobEvent.Action, workerEvent.JobEvent.Id.String())
			}
		case taskEvent := <-p.newTask:
			err := p.repo.EnqueueEncodeJob(ctx, taskEvent.Event)
			if err != nil {
				taskEvent.ControlChan <- err
				helper.Infof("failed publish job %s: %v", taskEvent.Event.Id.String(), err)
			} else {
				helper.Infof("published job %s", taskEvent.Event.Id.String())
			}
			close(taskEvent.ControlChan)
		}
	}
}

func (p *PostgresBrokerServer) taskEventProcessor(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events, err := p.repo.DequeueTaskEvents(ctx, constants.TaskEventDequeueLimit)
			if err != nil {
				helper.Errorf("failed to dequeue task events: %v", err)
				continue
			}

			for _, event := range events {
				err = p.repo.WithTransaction(ctx, func(ctx context.Context, tx repository.Repository) error {
					err = tx.ProcessEvent(ctx, event)
					if err != nil {
						return err
					}
					for _, consumer := range p.taskEventConsumers {
						select {
						case consumer <- event:
						default:
						}
					}
					return nil
				})
				if err != nil {
					helper.Errorf("taskencode event error: %s", err.Error())
					if event.EventType != model.PingEvent {
						helper.Debugf("failed event: %+v", event)
					}
				}
			}
		}
	}
}

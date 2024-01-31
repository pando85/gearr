package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
	"transcoder/broker"
	"transcoder/model"
	"transcoder/server/repository"

	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type BrokerServer interface {
	Run(wg *sync.WaitGroup, ctx context.Context)
	PublishJobRequest(request *model.TaskEncode) error
	PublishJobEvent(jobEvent *model.JobEvent, workerQueue string)
	ReceiveJobEvent() <-chan *model.TaskEvent
}

type RabbitMQServer struct {
	broker.Config
	connection         *rabbitmq.Connection
	repo               repository.Repository
	newTask            chan *model.ControlEvent
	newWorkerEvent     chan *model.JobEventQueue
	taskEventConsumers []chan *model.TaskEvent
}

func (Q *RabbitMQServer) conn() (*rabbitmq.Connection, error) {
	conn, err := rabbitmq.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", Q.User, Q.Password, Q.Host, Q.Port))

	return conn, err

}

func NewBrokerServerRabbit(config broker.Config, repo repository.Repository) (*RabbitMQServer, error) {
	queueRabbit := &RabbitMQServer{
		Config:         config,
		repo:           repo,
		newTask:        make(chan *model.ControlEvent),
		newWorkerEvent: make(chan *model.JobEventQueue),
	}
	return queueRabbit, nil
}

func (Q *RabbitMQServer) Run(wg *sync.WaitGroup, ctx context.Context) {
	log.Info("starting broker")
	Q.start(ctx)
	log.Info("started broker")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("stopping broker")
		Q.stop()
		wg.Done()
	}()
}

func (Q *RabbitMQServer) PublishJobRequest(taskRequest *model.TaskEncode) error {
	controlChan := make(chan interface{})
	Q.newTask <- &model.ControlEvent{
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
func (Q *RabbitMQServer) ReceiveJobEvent() <-chan *model.TaskEvent {
	tc := make(chan *model.TaskEvent, 100)
	Q.taskEventConsumers = append(Q.taskEventConsumers, tc)
	return tc
}
func (Q *RabbitMQServer) PublishJobEvent(jobEvent *model.JobEvent, workerQueue string) {
	jobEventQueue := &model.JobEventQueue{
		Queue:    workerQueue,
		JobEvent: jobEvent,
	}
	Q.newWorkerEvent <- jobEventQueue
}

func (Q *RabbitMQServer) start(ctx context.Context) {
	conn, err := Q.conn()
	if err != nil {
		log.Panic(err)
	}
	Q.connection = conn

	go Q.taskQueue(ctx)
	go Q.taskEventQueue(ctx)

}

func (Q *RabbitMQServer) stop() {

}

func (Q *RabbitMQServer) taskQueue(ctx context.Context) {
	taskChannel, err := Q.connection.Channel()
	if err != nil {
		log.Panic(err)
	}
	taskQueue, err := taskChannel.QueueDeclare(Q.TaskEncodeQueueName, true, false, false, false, nil)
	if err != nil {
		log.Panic(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case workerEvent := <-Q.newWorkerEvent:
			b, _ := json.Marshal(workerEvent.JobEvent)
			message := amqp.Publishing{
				ContentType: "text/plain",
				Type:        "JobEvent",
				Body:        b,
			}
			log.Infof("sending %s action for job %s", workerEvent.JobEvent.Action, workerEvent.JobEvent.Id.String())
			taskChannel.Publish("", workerEvent.Queue, false, false, message)
		case taskEvent := <-Q.newTask:
			b, err := json.Marshal(taskEvent.Event)
			if err != nil {
				taskEvent.ControlChan <- err
			}
			message := amqp.Publishing{
				ContentType: "text/plain",
				Body:        b,
			}
			if err := taskChannel.Publish("", taskQueue.Name, false, false, message); err != nil {
				taskEvent.ControlChan <- err
				log.Infof("failed publish job %s", taskEvent.Event.Id.String())
			} else {
				log.Infof("published job %s", taskEvent.Event.Id.String())
			}
			close(taskEvent.ControlChan)

		}
	}

}

func (Q *RabbitMQServer) taskEventQueue(ctx context.Context) {
	taskEventChannel, err := Q.connection.Channel()
	if err != nil {
		log.Panic(err)
	}
	taskEventsQueue, err := taskEventChannel.QueueDeclare(Q.TaskEventQueueName, true, false, false, false, nil)
	if err != nil {
		log.Panic(err)
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	taskEvents, err := taskEventChannel.Consume(taskEventsQueue.Name, fmt.Sprintf("%s-%d", "server", rnd.Intn(5000000)), false, true, false, false, nil)
	if err != nil {
		log.Panic(err)
	}
	for {
		select {
		case <-ctx.Done():
		case taskEventQueue := <-taskEvents:
			taskEvent := &model.TaskEvent{}
			err := json.Unmarshal(taskEventQueue.Body, taskEvent)
			if err != nil {
				log.Panic(err)
			}
			err = Q.repo.WithTransaction(ctx, func(ctx context.Context, tx repository.Repository) error {
				err = tx.ProcessEvent(ctx, taskEvent)
				if err != nil {
					return err
				}
				for _, consumer := range Q.taskEventConsumers {
					consumer <- taskEvent
				}

				err = taskEventQueue.Ack(false)
				if err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				taskEventQueue.Nack(false, false)
				log.Errorf("taskencode event error, requeued, with error: %s", err.Error())
				if taskEvent.EventType != model.PingEvent {
					b, _ := json.MarshalIndent(taskEvent, "", "\t")
					fmt.Println(string(b))
				}
			}
		}
	}
}

package queue

import (
	"context"
	"gearr/model"
	"gearr/server/repository"
	"sync"
)

type BrokerServer interface {
	Run(wg *sync.WaitGroup, ctx context.Context)
	PublishJobRequest(request *model.TaskEncode) error
	PublishJobEvent(jobEvent *model.JobEvent, workerQueue string)
	ReceiveJobEvent() <-chan *model.TaskEvent
}

func NewBrokerServer(repo repository.Repository) (*PostgresBrokerServer, error) {
	return NewBrokerServerPostgres(repo)
}

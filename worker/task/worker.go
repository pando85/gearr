package task

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"transcoder/model"
)

func NewWorkerClient(config Config, queue model.WorkerQueue) *WorkerRuntime {
	return &WorkerRuntime{
		config: config,
		queue:  queue,
	}
}

type WorkerRuntime struct {
	config  Config
	queue   model.WorkerQueue
	workers []model.QueueWorker
}

func (W *WorkerRuntime) Run(wg *sync.WaitGroup, ctx context.Context) {
	log.Info("Starting Worker Client...")
	W.start(ctx)
	log.Info("Started Worker Client...")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("Stopping Worker Client...")
		W.stop()
		wg.Done()
	}()
}
func (W *WorkerRuntime) start(ctx context.Context) {
	if W.config.Jobs.IsAccepted(model.EncodeJobType) {
		for i := 0; i < W.config.EncodeJobs; i++ {
			encodeWorker := NewEncodeWorker(ctx, W.config, fmt.Sprintf("%s-%d", model.EncodeJobType, i))
			W.workers = append(W.workers, encodeWorker)
			W.queue.RegisterWorker(encodeWorker)
			log.Infof("Initializing new %s worker name:%s", model.EncodeJobType, encodeWorker.GetID())
		}
	}
	if W.config.Jobs.IsAccepted(model.PGSToSrtJobType) {
		for i := 0; i < W.config.PgsJobs; i++ {
			pgsWorker := NewPGSWorker(ctx, W.config, fmt.Sprintf("%s-%d", model.PGSToSrtJobType, i))
			W.workers = append(W.workers, pgsWorker)
			W.queue.RegisterWorker(pgsWorker)
			log.Infof("Initializing new %s worker name:%s", model.PGSToSrtJobType, pgsWorker.GetID())
		}
	}
}

func (W *WorkerRuntime) stop() {
	log.Warnf("Stopping all Workers")
	for _, worker := range W.workers {
		worker.Cancel()
	}
}

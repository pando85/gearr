package task

import (
	"context"
	"fmt"
	"gearr/model"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
)

func NewWorkerClient(config Config, rabbit *RabbitMQClient, printer *ConsoleWorkerPrinter) *WorkerRuntime {
	return &WorkerRuntime{
		config:  config,
		rabbit:  rabbit,
		printer: printer,
	}
}

type WorkerRuntime struct {
	config       Config
	EncodeWorker *EncodeWorker
	PGSWorker    []*PGSWorker
	rabbit       *RabbitMQClient
	printer      *ConsoleWorkerPrinter
}

func (W *WorkerRuntime) Run(wg *sync.WaitGroup, ctx context.Context) {
	log.Info("starting worker client")
	W.start(ctx)
	log.Info("started worker client")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Info("stopping worker client")
		W.stop()
		wg.Done()
	}()
}
func (W *WorkerRuntime) start(ctx context.Context) {
	if W.config.Jobs.IsAccepted(model.EncodeJobType) {
		W.EncodeWorker = NewEncodeWorker(ctx, W.config, fmt.Sprintf("%s-%d", model.EncodeJobType, 1), W.printer)
		W.rabbit.RegisterEncodeWorker(W.EncodeWorker)
		W.EncodeWorker.Initialize()
		log.Info("initializing encode worker")

	}
	if W.config.Jobs.IsAccepted(model.PGSToSrtJobType) {
		for i := 0; i < runtime.NumCPU(); i++ {
			pgsWorker := NewPGSWorker(ctx, W.config, fmt.Sprintf("%s-%d", model.PGSToSrtJobType, i))
			log.Infof("initializing pgs worker %d", i)
			W.PGSWorker = append(W.PGSWorker, pgsWorker)
			W.rabbit.RegisterPGSWorker(pgsWorker)
		}
	}
}

func (W *WorkerRuntime) stop() {
	log.Warnf("stopping all workers")
	if W.EncodeWorker != nil {
		W.EncodeWorker.Cancel()
	}
	if W.PGSWorker != nil {
		for _, worker := range W.PGSWorker {
			worker.Cancel()
		}
	}
}

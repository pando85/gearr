package task

import (
	"context"
	"fmt"
	"gearr/helper"
	"gearr/model"
	"runtime"
	"sync"
)

type BrokerClient interface {
	model.Manager
	RegisterPGSWorker(worker *PGSWorker)
	RegisterEncodeWorker(worker *EncodeWorker)
	Run(wg *sync.WaitGroup, ctx context.Context)
}

func NewWorkerClient(config Config, brokerClient BrokerClient, printer *ConsoleWorkerPrinter) *WorkerRuntime {
	return &WorkerRuntime{
		config:       config,
		brokerClient: brokerClient,
		printer:      printer,
	}
}

type WorkerRuntime struct {
	config       Config
	EncodeWorker *EncodeWorker
	PGSWorker    []*PGSWorker
	brokerClient BrokerClient
	printer      *ConsoleWorkerPrinter
}

func (W *WorkerRuntime) Run(wg *sync.WaitGroup, ctx context.Context) {
	helper.Info("starting worker client")
	W.start(ctx)
	helper.Info("started worker client")
	wg.Add(1)
	go func() {
		<-ctx.Done()
		helper.Info("stopping worker client")
		W.stop()
		wg.Done()
	}()
}
func (W *WorkerRuntime) start(ctx context.Context) {
	if W.config.Jobs.IsAccepted(model.EncodeJobType) {
		W.EncodeWorker = NewEncodeWorker(ctx, W.config, fmt.Sprintf("%s-%d", model.EncodeJobType, 1), W.printer)
		W.brokerClient.RegisterEncodeWorker(W.EncodeWorker)
		W.EncodeWorker.Initialize()
		helper.Info("initializing encode worker")

	}
	if W.config.Jobs.IsAccepted(model.PGSToSrtJobType) {
		for i := 0; i < runtime.NumCPU(); i++ {
			pgsWorker := NewPGSWorker(ctx, W.config, fmt.Sprintf("%s-%d", model.PGSToSrtJobType, i))
			helper.Infof("initializing pgs worker %d", i)
			W.PGSWorker = append(W.PGSWorker, pgsWorker)
			W.brokerClient.RegisterPGSWorker(pgsWorker)
		}
	}
}

func (W *WorkerRuntime) stop() {
	helper.Warnf("stopping all workers")
	if W.EncodeWorker != nil {
		W.EncodeWorker.Cancel()
	}
	if W.PGSWorker != nil {
		for _, worker := range W.PGSWorker {
			worker.Cancel()
		}
	}
}

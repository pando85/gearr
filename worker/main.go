package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	pflag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"transcoder/broker"
	"transcoder/cmd"
	"transcoder/helper"
	"transcoder/model"
	"transcoder/worker/queue"
	"transcoder/worker/task"
	"transcoder/worker/update"
)

type CmdLineOpts struct {
	Broker broker.Config `mapstructure:"broker"`
	Worker task.Config   `mapstructure:"worker"`
}

var (
	opts CmdLineOpts
	ApplicationFileName string
)

func init() {

	hostname, err := os.Hostname()
	if err != nil {
		log.Panic(err)
	}

	cmd.BrokerFlags()
	pflag.Bool("worker.noUpdateMode", false, "Run as Updater")
	pflag.String("worker.temporalPath", os.TempDir(), "Path used for temporal data")
	pflag.String("worker.name", hostname, "Worker Name used for statistics")
	pflag.Int("worker.threads", runtime.NumCPU(), "Worker Threads")
	pflag.StringSlice("worker.acceptedJobs", []string{"encode"}, "type of jobs this Worker will accept: encode,pgsTosrt")
	pflag.Int("worker.encodeJobs", 1, "Worker Encode Jobs in parallel")
	pflag.Int("worker.pgsJobs", 0, "Worker PGS Jobs in parallel")
	pflag.Int("worker.priority", 3, "Only Accept Jobs of priority X( Priority 1= <30 Min, 2=<60 Min,3=<2 Hour,4=<3 Hour,5=>3 Hour,6-9 Manual High Priority tasks")
	pflag.String("worker.dotnetPath", "dotnet", "dotnet path")
	pflag.String("worker.pgsToSrtDLLPath", "./PgsToSrt.dll", "PGSToSrt.dll path")
	pflag.String("worker.tesseractDataPath", "./tessdata", "tesseract data path")
	pflag.Var(&opts.Worker.StartAfter,"worker.startAfter",  "Accept jobs only After HH:mm")
	pflag.Var(&opts.Worker.StopAfter,"worker.stopAfter",  "Stop Accepting new Jobs after HH:mm")
	pflag.Usage = usage

	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/transcoderw/")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TR")
	err = viper.ReadInConfig()
	if err!=nil {
		switch err.(type){
		case viper.ConfigFileNotFoundError:
		default:
			log.Panic(err)
		}
	}
	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
	viperDecoder := viper.DecodeHook(func(source reflect.Type,target  reflect.Type, data interface{}) (interface{}, error){
		if source.Kind() != reflect.String {
			return data, nil
		}
		timeHourMinute := task.TimeHourMinute{}
		if target == reflect.TypeOf(timeHourMinute) {
			timeHourMinute.Set(data.(string))
			return timeHourMinute, nil
		}
		return data,nil
	})
	err = viper.Unmarshal(&opts,viperDecoder)
	if err != nil {
		log.Panic(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTION]...\n", os.Args[0])
	pflag.PrintDefaults()
	os.Exit(0)
}


func main() {
	log.SetLevel(log.DebugLevel)
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		wg.Add(1)
		shutdownHandler(ctx, sigs, cancel)
		wg.Done()
	}()
	helper.ApplicationFileName= ApplicationFileName
	if !opts.Worker.NoUpdateMode {
		updater := update.NewUpdater()
		updater.Run(wg,ctx)
	}else{
		//Prepare work environment
		prepareWorkerEnvironment(ctx,assets,&opts.Worker.Jobs)

		//BrokerClient System
		broker := queue.NewBrokerClientRabbit(opts.Broker, opts.Worker)
		broker.Run(wg, ctx)

		worker := task.NewWorkerClient(opts.Worker, broker)
		worker.Run(wg, ctx)
	}
	wg.Wait()
}


func shutdownHandler(ctx context.Context, sigs chan os.Signal, cancel context.CancelFunc) {
	select {
	case <-sigs:
		cancel()
		log.Info("Termination Signal Detected...")
	}

	signal.Stop(sigs)
}

func prepareWorkerEnvironment(ctx context.Context,assets http.FileSystem,acceptedJobs *task.AcceptedJobs) {
	log.Infof("Initializing Environment...")
	if acceptedJobs.IsAccepted(model.EncodeJobType) {
		if err:=helper.DesembedFSFFProbe(assets);err!=nil {
			panic(err)
		}

		if err:=helper.DesembedFFmpeg(assets);err!=nil {
			panic(err)
		}

		if err:=helper.DesembedMKVExtract(assets);err!=nil {
			panic(err)
		}
	}
}


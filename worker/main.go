package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"transcoder/broker"
	"transcoder/cmd"
	"transcoder/helper"
	"transcoder/worker/task"

	log "github.com/sirupsen/logrus"
	pflag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type CmdLineOpts struct {
	Broker   broker.Config `mapstructure:"broker"`
	Worker   task.Config   `mapstructure:"worker"`
	LogLevel string        `mapstructure:"log-level"`
}

var (
	opts                CmdLineOpts
	ApplicationFileName string
)

func init() {

	hostname, err := os.Hostname()
	if err != nil {
		log.Panic(err)
	}

	cmd.BrokerFlags()
	cmd.LogLevelFlags()
	pflag.String("worker.temporalPath", os.TempDir(), "Path used for temporal data")
	pflag.String("worker.name", hostname, "Worker Name used for statistics")
	pflag.Int("worker.threads", runtime.NumCPU(), "Worker Threads")
	pflag.StringSlice("worker.acceptedJobs", []string{"encode"}, "type of jobs this Worker will accept: encode,pgstosrt")
	pflag.Int("worker.maxPrefetchJobs", 1, "Maximum number of jobs to prefetch")
	pflag.Int("worker.encodeJobs", 1, "Worker Encode Jobs in parallel")
	pflag.Int("worker.pgsJobs", 0, "Worker PGS Jobs in parallel")
	pflag.String("worker.dotnetPath", "/usr/bin/dotnet", "dotnet path")
	pflag.String("worker.pgsToSrtDLLPath", "/app/PgsToSrt.dll", "PGSToSrt.dll path")
	pflag.String("worker.tesseractDataPath", "/tessdata", "tesseract data path (https://github.com/tesseract-ocr/tessdata/)")
	pflag.Var(&opts.Worker.StartAfter, "worker.startAfter", "Accept jobs only After HH:mm")
	pflag.Var(&opts.Worker.StopAfter, "worker.stopAfter", "Stop Accepting new Jobs after HH:mm")

	pflag.Usage = usage

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	viper.SetConfigType("yaml")
	configFilePath := os.Getenv("CONFIG_PATH")

	if configFilePath == "" {
		configFilePath = "/app/config-worker.yaml"
	}

	viper.SetConfigFile(configFilePath)

	err = viper.ReadInConfig()
	if err != nil {
		log.Warnf("no config file found")
	}

	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
	viperDecoder := viper.DecodeHook(func(source reflect.Type, target reflect.Type, data interface{}) (interface{}, error) {
		if source.Kind() != reflect.String {
			return data, nil
		}
		timeHourMinute := task.TimeHourMinute{}
		if target == reflect.TypeOf(timeHourMinute) {
			timeHourMinute.Set(data.(string))
			return timeHourMinute, nil
		}
		return data, nil
	})
	err = viper.Unmarshal(&opts, viperDecoder)
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
	helper.SetLogLevel(opts.LogLevel)
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		wg.Add(1)
		shutdownHandler(ctx, sigs, cancel)
		wg.Done()
	}()
	helper.ApplicationFileName = ApplicationFileName
	log.Debugf("%+v", opts)

	printer := task.NewConsoleWorkerPrinter()

	//BrokerClient System
	broker := task.NewBrokerClientRabbit(opts.Broker, opts.Worker, printer)
	broker.Run(wg, ctx)

	worker := task.NewWorkerClient(opts.Worker, broker, printer)
	worker.Run(wg, ctx)

	wg.Wait()
}

func shutdownHandler(ctx context.Context, sigs chan os.Signal, cancel context.CancelFunc) {
	select {
	case <-sigs:
		cancel()
		log.Info("termination signal detected")
	}

	signal.Stop(sigs)
}

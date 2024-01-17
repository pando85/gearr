package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"
	"transcoder/broker"
	"transcoder/cmd"
	"transcoder/helper"
	"transcoder/server/queue"
	"transcoder/server/repository"
	"transcoder/server/scheduler"
	"transcoder/server/web"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type CmdLineOpts struct {
	Broker    broker.Config              `mapstructure:"broker"`
	Database  repository.SQLServerConfig `mapstructure:"database"`
	LogLevel  string                     `mapstructure:"loglevel"`
	Scheduler scheduler.SchedulerConfig  `mapstructure:"scheduler"`
	Web       web.WebServerConfig        `mapstructure:"web"`
}

var (
	opts                CmdLineOpts
	ApplicationFileName string
)

func init() {
	cmd.BrokerFlags()
	cmd.DatabaseFlags()
	cmd.LogLevelFlags()
	cmd.SchedulerFlags()
	cmd.WebFlags()

	pflag.Usage = usage

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetConfigType("yaml")

	configFilePath := os.Getenv("CONFIG_PATH")

	if configFilePath == "" {
		configFilePath = "/app/config.yaml"
	}

	viper.SetConfigFile(configFilePath)

	err := viper.ReadInConfig()
	if err != nil {
		log.Warnf("no config file found")
	}

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	urlAndDurationDecoder := viper.DecodeHook(func(source reflect.Type, target reflect.Type, data interface{}) (interface{}, error) {
		if source.Kind() != reflect.String {
			return data, nil
		}
		if target == reflect.TypeOf(url.URL{}) {
			url, err := url.Parse(data.(string))
			return url, err
		} else if target == reflect.TypeOf(time.Duration(5)) {
			return time.ParseDuration(data.(string))
		}
		return data, nil

	})
	err = viper.Unmarshal(&opts, urlAndDurationDecoder)
	if err != nil {
		log.Panic(err)
	}

	//Fix Paths
	opts.Scheduler.DownloadPath = filepath.Clean(opts.Scheduler.DownloadPath)
	opts.Scheduler.UploadPath = filepath.Clean(opts.Scheduler.UploadPath)
	helper.CheckPath(opts.Scheduler.DownloadPath)
	helper.CheckPath(opts.Scheduler.UploadPath)
	/*
	   scheduleTimeDuration, err := time.ParseDuration(opts.ScheduleTime)

	   	if err!=nil {
	   		log.Panic(err)
	   	}

	   jobTimeout, err := time.ParseDuration(opts.JobTimeout)

	   	if err!=nil {
	   		log.Panic(err)
	   	}

	   opts.Scheduler.ScheduleTime = scheduleTimeDuration
	   opts.Scheduler.JobTimeout = jobTimeout
	*/
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
		shutdownHandler(ctx, sigs, cancel)
		wg.Done()
	}()
	//Prepare resources
	log.Infof("preparing to runwithcontext")

	var repo repository.Repository
	repo, err := repository.NewSQLRepository(opts.Database)
	if err != nil {
		log.Panic(err)
	}
	err = repo.Initialize(ctx)
	if err != nil {
		log.Panic(err)
	}

	//BrokerServer System
	broker, err := queue.NewBrokerServerRabbit(opts.Broker, repo)
	if err != nil {
		log.Panic(err)
	}
	broker.Run(wg, ctx)

	//Scheduler
	scheduler, err := scheduler.NewScheduler(opts.Scheduler, repo, broker)
	if err != nil {
		log.Panic(err)
	}
	scheduler.Run(wg, ctx)

	//Web Server
	var webServer *web.WebServer
	webServer = web.NewWebServer(opts.Web, scheduler)
	webServer.Run(wg, ctx)
	wg.Wait()
}

func shutdownHandler(ctx context.Context, sigs chan os.Signal, cancel context.CancelFunc) {
	select {
	case <-ctx.Done():
		log.Info("termination signal detected")
	case <-sigs:
		cancel()
		log.Info("termination signal detected")
	}

	signal.Stop(sigs)
}

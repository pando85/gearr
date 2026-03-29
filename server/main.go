package main

import (
	"context"
	"fmt"
	"gearr/cmd"
	"gearr/helper"
	"gearr/model"
	"gearr/server/auth"
	"gearr/server/queue"
	"gearr/server/repository"
	libscanner "gearr/server/scanner"
	"gearr/server/scheduler"
	"gearr/server/watcher"
	"gearr/server/web"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type CmdLineOpts struct {
	Database  repository.SQLServerConfig `mapstructure:"database"`
	LogLevel  string                     `mapstructure:"log-level"`
	Scheduler scheduler.SchedulerConfig  `mapstructure:"scheduler"`
	Web       web.WebServerConfig        `mapstructure:"web"`
	Watcher   watcher.Config             `mapstructure:"watcher"`
	Scanner   model.ScannerConfig        `mapstructure:"scanner"`
	Priority  model.PriorityConfig       `mapstructure:"priority"`
	Webhook   model.WebhookConfig        `mapstructure:"webhook"`
	Auth      auth.AuthConfig            `mapstructure:"auth"`
}

var (
	opts CmdLineOpts
)

func init() {
	cmd.DatabaseFlags()
	cmd.LogLevelFlags()
	cmd.SchedulerFlags()
	cmd.WebFlags()
	cmd.WatcherFlags()
	cmd.ScannerFlags()
	cmd.PriorityFlags()
	cmd.WebhookFlags()
	cmd.AuthFlags()

	pflag.Usage = usage

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.SetConfigType("yaml")

	configFilePath := os.Getenv("CONFIG_PATH")

	if configFilePath == "" {
		configFilePath = "/app/config.yaml"
	}

	viper.SetConfigFile(configFilePath)

	err := viper.ReadInConfig()
	if err != nil {
		helper.Warnf("no config file found")
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
		helper.Panic(err)
	}

	opts.Scheduler.DownloadPath = filepath.Clean(opts.Scheduler.DownloadPath)
	opts.Scheduler.UploadPath = filepath.Clean(opts.Scheduler.UploadPath)
	helper.CheckPath(opts.Scheduler.DownloadPath)
	helper.CheckPath(opts.Scheduler.UploadPath)

	opts.Watcher.DownloadPath = opts.Scheduler.DownloadPath
	opts.Watcher.MinFileSize = opts.Scheduler.MinFileSize
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
	wg.Add(1)
	go func() {
		shutdownHandler(ctx, sigs, cancel)
		wg.Done()
	}()
	helper.Infof("preparing to runwithcontext")

	var repo repository.Repository
	repo, err := repository.NewSQLRepository(opts.Database)
	if err != nil {
		helper.Panic(err)
	}
	err = repo.Initialize(ctx)
	if err != nil {
		helper.Panic(err)
	}

	broker, err := queue.NewBrokerServer(repo)
	if err != nil {
		helper.Panic(err)
	}
	broker.Run(wg, ctx)

	scheduler, err := scheduler.NewScheduler(opts.Scheduler, repo, broker)
	if err != nil {
		helper.Panic(err)
	}
	scheduler.Run(wg, ctx)

	watcherSvc, err := watcher.NewWatcher(opts.Watcher, scheduler, repo)
	if err != nil {
		helper.Panic(err)
	}
	watcherSvc.Run(wg, ctx)

	var libScanner *libscanner.Scanner
	if opts.Scanner.Enabled && len(opts.Scanner.Paths) > 0 {
		libScanner = libscanner.NewScanner(opts.Scanner, repo, scheduler)
		libScanner.Run(wg, ctx)
		helper.Info("library scanner started")
	}

	authService, err := auth.NewAuthService(opts.Auth, repo)
	if err != nil {
		helper.Panic(err)
	}

	if opts.Auth.Token == "" && opts.Web.Token != "" {
		opts.Auth.Token = opts.Web.Token
		authService.SetStaticToken(opts.Web.Token)
	}

	var webServer *web.WebServer
	opts.Web.WebhookConfig = &opts.Webhook
	opts.Web.AuthConfig = &opts.Auth
	webServer = web.NewWebServer(opts.Web, scheduler, watcherSvc, libScanner, repo, authService)
	webServer.Run(wg, ctx)
	wg.Wait()
}

func shutdownHandler(ctx context.Context, sigs chan os.Signal, cancel context.CancelFunc) {
	select {
	case <-ctx.Done():
		helper.Info("termination signal detected")
	case <-sigs:
		cancel()
		helper.Info("termination signal detected")
	}

	signal.Stop(sigs)
}

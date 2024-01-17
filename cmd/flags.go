package cmd

import (
	"time"

	"github.com/spf13/pflag"
)

func BrokerFlags() {
	pflag.String("broker.host", "localhost", "Broker Host")
	pflag.Int("broker.port", 5672, "WebServer Port")
	pflag.String("broker.user", "broker", "Broker User")
	pflag.String("broker.password", "broker", "Broker User")
	pflag.String("broker.taskEncodeQueue", "tasks", "Broker tasks queue name")
	pflag.String("broker.taskPGSQueue", "tasks_pgstosrt", "Broker tasks pgstosrt queue name")
	pflag.String("broker.eventQueue", "task_events", "Broker tasks events queue name")
}

func DatabaseFlags() {
	pflag.String("database.Driver", "postgres", "DB Driver")
	pflag.String("database.Host", "localhost", "DB Host")
	pflag.Int("database.port", 5432, "DB Port")
	pflag.String("database.User", "postgres", "DB User")
	pflag.String("database.Password", "postgres", "DB Password")
	pflag.String("database.Database", "transcoder", "DB Database")
	pflag.String("database.SSLMode", "disable", "DB Scheme")
}

func LogLevelFlags() {
	pflag.String("log-level", "info", "Set the log level (debug, info, warning, error, fatal)")
}

func SchedulerFlags() {
	pflag.String("scheduler.domain", "http://localhost:8080", "Base domain where workers will try to download upload videos")
	pflag.Duration("scheduler.scheduleTime", time.Minute*5, "Execute the scheduling loop every X seconds")
	pflag.Duration("scheduler.jobTimeout", time.Hour*24, "Requeue jobs that are running for more than X minutes")
	pflag.String("scheduler.downloadPath", "/data/current", "Download path")
	pflag.String("scheduler.uploadPath", "/data/processed", "Upload path")
	pflag.Int64("scheduler.minFileSize", 1e+8, "Min File Size")
}

func WebFlags() {
	pflag.Int("web.port", 8080, "WebServer Port")
	pflag.String("web.token", "admin", "WebServer Port")
}

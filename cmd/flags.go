package cmd

import (
	"time"

	"github.com/spf13/pflag"
)

func DatabaseFlags() {
	pflag.String("database.driver", "pgx", "DB Driver")
	pflag.String("database.host", "localhost", "DB Host")
	pflag.Int("database.port", 5432, "DB Port")
	pflag.String("database.user", "postgres", "DB User")
	pflag.String("database.password", "postgres", "DB Password")
	pflag.String("database.database", "gearr", "DB Database")
	pflag.String("database.sslmode", "disable", "DB Scheme")
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

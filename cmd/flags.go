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

func WatcherFlags() {
	pflag.StringSlice("scheduler.watchPaths", []string{}, "Comma-separated paths to watch for new video files")
	pflag.Bool("scheduler.watchEnabled", false, "Enable folder watcher for auto-detecting new video files")
	pflag.Duration("scheduler.watchDebounce", 2*time.Second, "Debounce time for file events")
	pflag.StringSlice("scheduler.watchPatterns", []string{"*.mkv", "*.mp4", "*.avi", "*.mov", "*.wmv", "*.flv"}, "File patterns to watch")
}

func ScannerFlags() {
	pflag.Bool("scanner.enabled", false, "Enable library scanner")
	pflag.Duration("scanner.interval", time.Hour*24, "Scan interval for library scanner")
	pflag.Int64("scanner.minFileSize", 1e+8, "Minimum file size for library scanner (bytes)")
	pflag.StringSlice("scanner.paths", []string{}, "Paths to scan for video files")
	pflag.StringSlice("scanner.fileExtensions", []string{}, "File extensions to scan (default: common video extensions)")
}

func PriorityFlags() {
	pflag.Bool("priority.priorityBySize", false, "Prioritize larger files (larger files = higher priority)")
	pflag.Bool("priority.priorityByAge", false, "Prioritize older files (older files = higher priority)")
	pflag.Int("priority.defaultPriority", 5, "Default priority for jobs (1-10, where 10 is highest)")
}

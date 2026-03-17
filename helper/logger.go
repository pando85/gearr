package helper

import (
	"fmt"
	"log/slog"
	"os"
)

var (
	logger     *slog.Logger
	logLevel   = slog.LevelInfo
	logHandler slog.Handler
)

func init() {
	logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	logger = slog.New(logHandler)
	slog.SetDefault(logger)
}

func SetLogLevel(level string) {
	switch level {
	case "debug", "Debug":
		logLevel = slog.LevelDebug
	case "info", "Info":
		logLevel = slog.LevelInfo
	case "warning", "Warning", "warn", "Warn":
		logLevel = slog.LevelWarn
	case "error", "Error":
		logLevel = slog.LevelError
	case "fatal", "Fatal":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
		Warnf("invalid log level '%s', defaulting to 'info'", level)
	}
	logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	logger = slog.New(logHandler)
	slog.SetDefault(logger)
}

func Debug(args ...interface{}) {
	logger.Debug(fmt.Sprint(args...))
}

func Debugf(format string, args ...interface{}) {
	logger.Debug(fmt.Sprintf(format, args...))
}

func Info(args ...interface{}) {
	logger.Info(fmt.Sprint(args...))
}

func Infof(format string, args ...interface{}) {
	logger.Info(fmt.Sprintf(format, args...))
}

func Warn(args ...interface{}) {
	logger.Warn(fmt.Sprint(args...))
}

func Warnf(format string, args ...interface{}) {
	logger.Warn(fmt.Sprintf(format, args...))
}

func Error(args ...interface{}) {
	logger.Error(fmt.Sprint(args...))
}

func Errorf(format string, args ...interface{}) {
	logger.Error(fmt.Sprintf(format, args...))
}

func Panic(args ...interface{}) {
	msg := fmt.Sprint(args...)
	logger.Error(msg)
	panic(msg)
}

func Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logger.Error(msg)
	panic(msg)
}

func Fatal(args ...interface{}) {
	logger.Error(fmt.Sprint(args...))
	os.Exit(1)
}

func Fatalf(format string, args ...interface{}) {
	logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func WithField(key string, value interface{}) *slog.Logger {
	return logger.With(key, value)
}

func WithFields(fields map[string]interface{}) *slog.Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return logger.With(args...)
}

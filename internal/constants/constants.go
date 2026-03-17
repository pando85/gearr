package constants

import "time"

const (
	DBMaxOpenConns    = 5
	DBMaxIdleConns    = 5
	DBConnMaxLifetime = 5 * time.Minute
)

const (
	RetryDelay = 5 * time.Second

	DownloadRetryAttempts = 180
	UploadRetryAttempts   = 17280
	ChecksumRetryAttempts = 10
)

const (
	ChannelBufferSize = 100
)

const (
	IOBufferSize = 128 * 1024
)

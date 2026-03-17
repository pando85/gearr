package constants

import "time"

const (
	DBMaxOpenConns    = 5
	DBMaxIdleConns    = 5
	DBConnMaxLifetime = 5 * time.Minute
)

const (
	DownloadRetryAttempts = 180
	DownloadRetryDelay    = 5 * time.Second
	UploadRetryAttempts   = 17280
	ChecksumRetryAttempts = 10
)

const (
	ChannelBufferSize     = 100
	TaskEventDequeueLimit = 10
)

const (
	IOBufferSize = 128 * 1024
)

package constants

import "time"

const (
	DBMaxOpenConns    = 5
	DBMaxIdleConns    = 5
	DBConnMaxLifetime = 5 * time.Minute
)

const (
	ChannelBufferSize = 100
)

const (
	TaskEventDequeueLimit = 10
)

const (
	IOBufferSize = 128 * 1024
)

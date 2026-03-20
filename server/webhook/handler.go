package webhook

import (
	"time"
)

type WebhookEventType string

const (
	EventDownload     WebhookEventType = "Download"
	EventGrab         WebhookEventType = "Grab"
	EventRename       WebhookEventType = "Rename"
	EventMovieDelete  WebhookEventType = "MovieDelete"
	EventSeriesDelete WebhookEventType = "SeriesDelete"
	EventHealth       WebhookEventType = "Health"
	EventTest         WebhookEventType = "Test"
	EventApplication  WebhookEventType = "ApplicationUpdate"
)

type WebhookSource string

const (
	SourceRadarr WebhookSource = "radarr"
	SourceSonarr WebhookSource = "sonarr"
)

type WebhookHandler interface {
	HandleEvent(event *WebhookEvent) (*WebhookResponse, error)
	SupportedEvents() []WebhookEventType
	SupportedSources() []WebhookSource
}

type WebhookEvent struct {
	EventType    WebhookEventType `json:"event_type"`
	Source       WebhookSource    `json:"source"`
	Timestamp    time.Time        `json:"timestamp"`
	Instance     string           `json:"instance,omitempty"`
	RawPayload   []byte           `json:"raw_payload,omitempty"`
	Movie        *MoviePayload    `json:"movie,omitempty"`
	Series       *SeriesPayload   `json:"series,omitempty"`
	Episode      *EpisodePayload  `json:"episode,omitempty"`
	Release      *ReleasePayload  `json:"release,omitempty"`
	Health       *HealthPayload   `json:"health,omitempty"`
	DownloadID   string           `json:"download_id,omitempty"`
	DeletedFiles []string         `json:"deleted_files,omitempty"`
}

type WebhookResponse struct {
	Success    bool     `json:"success"`
	Message    string   `json:"message,omitempty"`
	Errors     []string `json:"errors,omitempty"`
	JobID      string   `json:"job_id,omitempty"`
	Processed  bool     `json:"processed"`
	Skipped    bool     `json:"skipped"`
	SkipReason string   `json:"skip_reason,omitempty"`
}

type MoviePayload struct {
	ID             int        `json:"id,omitempty"`
	Title          string     `json:"title,omitempty"`
	OriginalTitle  string     `json:"original_title,omitempty"`
	Year           int        `json:"year,omitempty"`
	IMDBID         string     `json:"imdb_id,omitempty"`
	TMDBID         int        `json:"tmdb_id,omitempty"`
	Overview       string     `json:"overview,omitempty"`
	FolderPath     string     `json:"folder_path,omitempty"`
	FilePath       string     `json:"file_path,omitempty"`
	Quality        string     `json:"quality,omitempty"`
	QualityVersion int        `json:"quality_version,omitempty"`
	ReleaseDate    string     `json:"release_date,omitempty"`
	Size           int64      `json:"size,omitempty"`
	Codec          string     `json:"codec,omitempty"`
	MediaInfo      *MediaInfo `json:"media_info,omitempty"`
	CustomFormats  []string   `json:"custom_formats,omitempty"`
	Languages      []string   `json:"languages,omitempty"`
	SceneName      string     `json:"scene_name,omitempty"`
	IndexerFlags   int        `json:"indexer_flags,omitempty"`
	ReleaseGroup   string     `json:"release_group,omitempty"`
}

type SeriesPayload struct {
	ID            int    `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	OriginalTitle string `json:"original_title,omitempty"`
	Year          int    `json:"year,omitempty"`
	IMDBID        string `json:"imdb_id,omitempty"`
	TVDBID        int    `json:"tvdb_id,omitempty"`
	TVRageID      int    `json:"tvrage_id,omitempty"`
	Overview      string `json:"overview,omitempty"`
	FolderPath    string `json:"folder_path,omitempty"`
	Network       string `json:"network,omitempty"`
	Status        string `json:"status,omitempty"`
	AirTime       string `json:"air_time,omitempty"`
}

type EpisodePayload struct {
	ID             int        `json:"id,omitempty"`
	EpisodeNumber  int        `json:"episode_number,omitempty"`
	SeasonNumber   int        `json:"season_number,omitempty"`
	Title          string     `json:"title,omitempty"`
	Overview       string     `json:"overview,omitempty"`
	AirDate        string     `json:"air_date,omitempty"`
	AirDateUTC     string     `json:"air_date_utc,omitempty"`
	FilePath       string     `json:"file_path,omitempty"`
	Quality        string     `json:"quality,omitempty"`
	QualityVersion int        `json:"quality_version,omitempty"`
	Size           int64      `json:"size,omitempty"`
	Codec          string     `json:"codec,omitempty"`
	MediaInfo      *MediaInfo `json:"media_info,omitempty"`
	SceneName      string     `json:"scene_name,omitempty"`
	ReleaseGroup   string     `json:"release_group,omitempty"`
	HasNfo         bool       `json:"has_nfo,omitempty"`
	Languages      []string   `json:"languages,omitempty"`
}

type ReleasePayload struct {
	Title           string `json:"title,omitempty"`
	Indexer         string `json:"indexer,omitempty"`
	Size            int64  `json:"size,omitempty"`
	Quality         string `json:"quality,omitempty"`
	QualityVersion  int    `json:"quality_version,omitempty"`
	ReleaseGroup    string `json:"release_group,omitempty"`
	SceneName       string `json:"scene_name,omitempty"`
	DownloadClient  string `json:"download_client,omitempty"`
	DownloadID      string `json:"download_id,omitempty"`
	TorrentInfoHash string `json:"torrent_info_hash,omitempty"`
	IsUpgrade       bool   `json:"is_upgrade,omitempty"`
}

type HealthPayload struct {
	Level   string `json:"level,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
	WikiURL string `json:"wiki_url,omitempty"`
}

type MediaInfo struct {
	VideoCodec        string        `json:"video_codec,omitempty"`
	VideoResolution   string        `json:"video_resolution,omitempty"`
	VideoBitDepth     int           `json:"video_bit_depth,omitempty"`
	VideoFramerate    string        `json:"video_framerate,omitempty"`
	AudioCodec        string        `json:"audio_codec,omitempty"`
	AudioChannels     int           `json:"audio_channels,omitempty"`
	AudioLanguages    []string      `json:"audio_languages,omitempty"`
	SubtitleLanguages []string      `json:"subtitle_languages,omitempty"`
	Duration          time.Duration `json:"duration,omitempty"`
	Width             int           `json:"width,omitempty"`
	Height            int           `json:"height,omitempty"`
}

type WebhookError struct {
	Code    WebhookErrorCode `json:"code"`
	Message string           `json:"message"`
	Cause   error            `json:"-"`
}

type WebhookErrorCode string

const (
	ErrInvalidPayload     WebhookErrorCode = "INVALID_PAYLOAD"
	ErrUnsupportedEvent   WebhookErrorCode = "UNSUPPORTED_EVENT"
	ErrUnsupportedSource  WebhookErrorCode = "UNSUPPORTED_SOURCE"
	ErrProcessingFailed   WebhookErrorCode = "PROCESSING_FAILED"
	ErrQueueFailed        WebhookErrorCode = "QUEUE_FAILED"
	ErrFileNotFound       WebhookErrorCode = "FILE_NOT_FOUND"
	ErrAlreadyTranscoding WebhookErrorCode = "ALREADY_TRANSCODING"
	ErrCodecFiltered      WebhookErrorCode = "CODEC_FILTERED"
	ErrSizeFiltered       WebhookErrorCode = "SIZE_FILTERED"
	ErrUnauthorized       WebhookErrorCode = "UNAUTHORIZED"
	ErrInternal           WebhookErrorCode = "INTERNAL_ERROR"
)

func (e *WebhookError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *WebhookError) Unwrap() error {
	return e.Cause
}

func NewWebhookError(code WebhookErrorCode, message string, cause error) *WebhookError {
	return &WebhookError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func IsWebhookError(err error) bool {
	_, ok := err.(*WebhookError)
	return ok
}

func GetWebhookErrorCode(err error) WebhookErrorCode {
	if webhookErr, ok := err.(*WebhookError); ok {
		return webhookErr.Code
	}
	return ErrInternal
}

func (e *WebhookEvent) IsDownloadEvent() bool {
	return e.EventType == EventDownload
}

func (e *WebhookEvent) IsGrabEvent() bool {
	return e.EventType == EventGrab
}

func (e *WebhookEvent) IsTestEvent() bool {
	return e.EventType == EventTest
}

func (e *WebhookEvent) IsHealthEvent() bool {
	return e.EventType == EventHealth
}

func (e *WebhookEvent) IsDeleteEvent() bool {
	return e.EventType == EventMovieDelete || e.EventType == EventSeriesDelete
}

func (e *WebhookEvent) IsFromRadarr() bool {
	return e.Source == SourceRadarr
}

func (e *WebhookEvent) IsFromSonarr() bool {
	return e.Source == SourceSonarr
}

func (e *WebhookEvent) GetFilePath() string {
	if e.Movie != nil && e.Movie.FilePath != "" {
		return e.Movie.FilePath
	}
	if e.Episode != nil && e.Episode.FilePath != "" {
		return e.Episode.FilePath
	}
	return ""
}

func (e *WebhookEvent) GetCodec() string {
	if e.Movie != nil && e.Movie.Codec != "" {
		return e.Movie.Codec
	}
	if e.Episode != nil && e.Episode.Codec != "" {
		return e.Episode.Codec
	}
	if e.Movie != nil && e.Movie.MediaInfo != nil && e.Movie.MediaInfo.VideoCodec != "" {
		return e.Movie.MediaInfo.VideoCodec
	}
	if e.Episode != nil && e.Episode.MediaInfo != nil && e.Episode.MediaInfo.VideoCodec != "" {
		return e.Episode.MediaInfo.VideoCodec
	}
	return ""
}

func (e *WebhookEvent) GetSize() int64 {
	if e.Movie != nil {
		return e.Movie.Size
	}
	if e.Episode != nil {
		return e.Episode.Size
	}
	return 0
}

func (e *WebhookEvent) GetTitle() string {
	if e.Movie != nil {
		return e.Movie.Title
	}
	if e.Series != nil {
		return e.Series.Title
	}
	return ""
}

func NewSuccessResponse(message string) *WebhookResponse {
	return &WebhookResponse{
		Success:   true,
		Message:   message,
		Processed: true,
	}
}

func NewErrorResponse(errors []string) *WebhookResponse {
	return &WebhookResponse{
		Success:   false,
		Errors:    errors,
		Processed: false,
	}
}

func NewSkippedResponse(reason string) *WebhookResponse {
	return &WebhookResponse{
		Success:    true,
		Processed:  false,
		Skipped:    true,
		SkipReason: reason,
	}
}

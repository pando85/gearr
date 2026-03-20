package webhook

import "time"

type Source string

const (
	SourceRadarr Source = "radarr"
	SourceSonarr Source = "sonarr"
)

type EventType string

const (
	EventDownload          EventType = "download"
	EventGrab              EventType = "grab"
	EventRename            EventType = "rename"
	EventMovieDelete       EventType = "movie_delete"
	EventSeriesDelete      EventType = "series_delete"
	EventEpisodeFileDelete EventType = "episodefile_delete"
	EventTest              EventType = "test"
)

type WebhookPayload struct {
	SourceType Source      `json:"source_type"`
	EventType  EventType   `json:"event_type"`
	RawPayload interface{} `json:"raw_payload,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
}

type File struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Name         string `json:"name"`
	Size         int64  `json:"size,omitempty"`
	Quality      string `json:"quality,omitempty"`
}

type MediaInfo struct {
	Title    string `json:"title"`
	FilePath string `json:"file_path"`
}

type WebhookResult struct {
	Accepted   bool      `json:"accepted"`
	Files      []File    `json:"files,omitempty"`
	MediaInfo  MediaInfo `json:"media_info,omitempty"`
	SkipReason string    `json:"skip_reason,omitempty"`
	Error      string    `json:"error,omitempty"`
}

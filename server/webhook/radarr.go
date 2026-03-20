package webhook

import (
	"context"
	"time"
)

type RadarrHandler struct {
	*BaseHandler
}

func NewRadarrHandler() *RadarrHandler {
	return &RadarrHandler{
		BaseHandler: NewBaseHandler(SourceRadarr, []EventType{EventDownload, EventGrab, EventTest}),
	}
}

func (h *RadarrHandler) Parse(ctx context.Context, payload []byte) (*WebhookPayload, error) {
	var basePayload radarrBasePayload
	if err := h.ParseRawPayload(payload, &basePayload); err != nil {
		return nil, err
	}

	eventType := h.mapEventType(basePayload.EventType)

	result := &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  eventType,
		Timestamp:  time.Now(),
	}

	switch eventType {
	case EventDownload:
		var downloadPayload radarrDownloadPayload
		if err := h.ParseRawPayload(payload, &downloadPayload); err != nil {
			return nil, err
		}
		result.RawPayload = downloadPayload
	case EventGrab:
		var grabPayload radarrGrabPayload
		if err := h.ParseRawPayload(payload, &grabPayload); err != nil {
			return nil, err
		}
		result.RawPayload = grabPayload
	case EventTest:
		var testPayload radarrGrabPayload
		if err := h.ParseRawPayload(payload, &testPayload); err != nil {
			return nil, err
		}
		result.RawPayload = testPayload
	}

	return result, nil
}

func (h *RadarrHandler) Process(ctx context.Context, payload *WebhookPayload) (*WebhookResult, error) {
	switch payload.EventType {
	case EventTest:
		return &WebhookResult{Accepted: true}, nil
	case EventDownload:
		return h.processDownload(payload)
	case EventGrab:
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "grab events do not have downloaded files yet",
		}, nil
	default:
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "unsupported event type",
		}, nil
	}
}

func (h *RadarrHandler) processDownload(payload *WebhookPayload) (*WebhookResult, error) {
	downloadPayload, ok := payload.RawPayload.(radarrDownloadPayload)
	if !ok {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "invalid download payload",
		}, nil
	}

	if downloadPayload.MovieFile.Path == "" {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no file path in download payload",
		}, nil
	}

	files := []File{
		{
			Path:         downloadPayload.MovieFile.Path,
			RelativePath: downloadPayload.MovieFile.RelativePath,
			Name:         downloadPayload.MovieFile.SceneName,
			Size:         downloadPayload.MovieFile.Size,
			Quality:      downloadPayload.MovieFile.Quality,
		},
	}

	mediaInfo := MediaInfo{
		Title:    downloadPayload.Movie.Title,
		FilePath: downloadPayload.Movie.FolderPath,
	}

	return &WebhookResult{
		Accepted:  true,
		Files:     files,
		MediaInfo: mediaInfo,
	}, nil
}

func (h *RadarrHandler) mapEventType(eventType string) EventType {
	switch eventType {
	case "Download":
		return EventDownload
	case "Grab":
		return EventGrab
	case "Test":
		return EventTest
	case "MovieFileDelete":
		return EventMovieDelete
	case "MovieDelete":
		return EventMovieDelete
	case "Rename":
		return EventRename
	default:
		return EventDownload
	}
}

type radarrBasePayload struct {
	EventType string `json:"eventType"`
}

type radarrMovie struct {
	Id          int      `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	FilePath    string   `json:"filePath"`
	FolderPath  string   `json:"folderPath"`
	TmdbId      int      `json:"tmdbId"`
	ImdbId      string   `json:"imdbId"`
	ReleaseDate string   `json:"releaseDate"`
	Genres      []string `json:"genres"`
}

type radarrMovieFile struct {
	Id           int                       `json:"id"`
	RelativePath string                    `json:"relativePath"`
	Path         string                    `json:"path"`
	Quality      string                    `json:"quality"`
	Size         int64                     `json:"size"`
	SceneName    string                    `json:"sceneName"`
	MediaInfo    *radarrMovieFileMediaInfo `json:"mediaInfo,omitempty"`
}

type radarrMovieFileMediaInfo struct {
	AudioBitRate    string `json:"audioBitRate"`
	AudioChannels   string `json:"audioChannels"`
	AudioCodec      string `json:"audioCodec"`
	AudioLanguages  string `json:"audioLanguages"`
	AudioProfile    string `json:"audioProfile"`
	VideoBitDepth   string `json:"videoBitDepth"`
	VideoBitRate    string `json:"videoBitRate"`
	VideoCodec      string `json:"videoCodec"`
	VideoFps        string `json:"videoFps"`
	VideoProfile    string `json:"videoProfile"`
	VideoResolution string `json:"videoResolution"`
}

type radarrRelease struct {
	Indexer      string `json:"indexer"`
	Quality      string `json:"quality"`
	ReleaseGroup string `json:"releaseGroup"`
	ReleaseTitle string `json:"releaseTitle"`
	Size         int64  `json:"size"`
}

type radarrDownloadPayload struct {
	radarrBasePayload
	Movie          radarrMovie     `json:"movie"`
	MovieFile      radarrMovieFile `json:"movieFile"`
	IsUpgrade      bool            `json:"isUpgrade"`
	DownloadId     string          `json:"downloadId"`
	DownloadClient string          `json:"downloadClient"`
	Release        radarrRelease   `json:"release"`
}

type radarrGrabPayload struct {
	radarrBasePayload
	Movie          radarrMovie   `json:"movie"`
	Release        radarrRelease `json:"release"`
	DownloadId     string        `json:"downloadId"`
	DownloadClient string        `json:"downloadClient"`
}

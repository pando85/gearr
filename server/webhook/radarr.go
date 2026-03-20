package webhook

import (
	"context"
	"time"

	"gearr/helper"
)

type RadarrHandler struct {
	*BaseHandler
}

func NewRadarrHandler() *RadarrHandler {
	return &RadarrHandler{
		BaseHandler: NewBaseHandler(SourceRadarr, []EventType{
			EventDownload,
			EventGrab,
			EventRename,
			EventMovieDelete,
			EventTest,
		}),
	}
}

type RadarrWebhookPayload struct {
	EventType string `json:"eventType"`
	Movie     struct {
		Title      string `json:"title"`
		FolderPath string `json:"folderPath"`
		Path       string `json:"path"`
	} `json:"movie"`
	RemoteMovie struct {
		Title string `json:"title"`
	} `json:"remoteMovie"`
	MovieFile struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Size         int64  `json:"size"`
		Quality      string `json:"quality"`
	} `json:"movieFile"`
	DeletedFiles []struct {
		Path string `json:"path"`
	} `json:"deletedFiles"`
	RenamedMovieFiles []struct {
		PreviousPath string `json:"previousPath"`
		Path         string `json:"path"`
		RelativePath string `json:"relativePath"`
	} `json:"renamedMovieFiles"`
	IsUpgrade bool `json:"isUpgrade"`
}

func (h *RadarrHandler) Parse(ctx context.Context, payload []byte) (*WebhookPayload, error) {
	var radarrPayload RadarrWebhookPayload
	if err := h.ParseRawPayload(payload, &radarrPayload); err != nil {
		return nil, err
	}

	eventType := h.mapEventType(radarrPayload.EventType)

	return &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  eventType,
		RawPayload: radarrPayload,
		Timestamp:  time.Now(),
	}, nil
}

func (h *RadarrHandler) Process(ctx context.Context, payload *WebhookPayload) (*WebhookResult, error) {
	radarrPayload, ok := payload.RawPayload.(RadarrWebhookPayload)
	if !ok {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "invalid payload type",
		}, nil
	}

	switch payload.EventType {
	case EventDownload:
		return h.processDownload(&radarrPayload)
	case EventGrab:
		return h.processGrab(&radarrPayload)
	case EventRename:
		return h.processRename(&radarrPayload)
	case EventMovieDelete:
		return h.processMovieDelete(&radarrPayload)
	case EventTest:
		return h.processTest(&radarrPayload)
	default:
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "unsupported event type",
		}, nil
	}
}

func (h *RadarrHandler) processDownload(payload *RadarrWebhookPayload) (*WebhookResult, error) {
	if payload.MovieFile.Path == "" {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no movie file path in download event",
		}, nil
	}

	helper.Infof("Radarr download event: %s - %s", payload.Movie.Title, payload.MovieFile.RelativePath)

	return &WebhookResult{
		Accepted: true,
		Files: []File{
			{
				Path:         payload.MovieFile.Path,
				RelativePath: payload.MovieFile.RelativePath,
				Name:         payload.MovieFile.RelativePath,
				Size:         payload.MovieFile.Size,
				Quality:      payload.MovieFile.Quality,
			},
		},
		MediaInfo: MediaInfo{
			Title:    payload.Movie.Title,
			FilePath: payload.MovieFile.Path,
		},
	}, nil
}

func (h *RadarrHandler) processGrab(payload *RadarrWebhookPayload) (*WebhookResult, error) {
	helper.Debugf("Radarr grab event: %s", payload.Movie.Title)
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "grab event does not have completed download",
	}, nil
}

func (h *RadarrHandler) processRename(payload *RadarrWebhookPayload) (*WebhookResult, error) {
	var files []File
	for _, renamed := range payload.RenamedMovieFiles {
		files = append(files, File{
			Path:         renamed.Path,
			RelativePath: renamed.RelativePath,
		})
	}

	if len(files) == 0 {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no renamed files in rename event",
		}, nil
	}

	helper.Infof("Radarr rename event: %d files renamed", len(files))

	return &WebhookResult{
		Accepted: true,
		Files:    files,
		MediaInfo: MediaInfo{
			Title:    payload.Movie.Title,
			FilePath: payload.Movie.FolderPath,
		},
	}, nil
}

func (h *RadarrHandler) processMovieDelete(payload *RadarrWebhookPayload) (*WebhookResult, error) {
	helper.Infof("Radarr movie delete event: %s", payload.Movie.Title)
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "movie delete event - no action needed",
	}, nil
}

func (h *RadarrHandler) processTest(payload *RadarrWebhookPayload) (*WebhookResult, error) {
	helper.Info("Radarr test event received")
	return &WebhookResult{
		Accepted: true,
	}, nil
}

func (h *RadarrHandler) mapEventType(eventType string) EventType {
	switch eventType {
	case "Download":
		return EventDownload
	case "Grab":
		return EventGrab
	case "Rename":
		return EventRename
	case "MovieDelete":
		return EventMovieDelete
	case "Test":
		return EventTest
	default:
		return EventType(eventType)
	}
}

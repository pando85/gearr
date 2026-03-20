package webhook

import (
	"context"
	"encoding/json"
	"time"
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
		Title string `json:"title"`
		Path  string `json:"folderPath"`
	} `json:"movie"`
	MovieFile struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Size         int64  `json:"size"`
		Quality      string `json:"quality,omitempty"`
	} `json:"movieFile"`
	MovieFiles []struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Size         int64  `json:"size"`
	} `json:"movieFiles"`
	RenamedMovieFiles []struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"previousPath"`
	} `json:"renamedMovieFiles"`
}

func (h *RadarrHandler) Parse(ctx context.Context, payload []byte) (*WebhookPayload, error) {
	var rawPayload RadarrWebhookPayload
	if err := h.ParseRawPayload(payload, &rawPayload); err != nil {
		return nil, err
	}

	eventType := h.mapEventType(rawPayload.EventType)

	return &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  eventType,
		RawPayload: rawPayload,
		Timestamp:  time.Now(),
	}, nil
}

func (h *RadarrHandler) Process(ctx context.Context, payload *WebhookPayload) (*WebhookResult, error) {
	rawPayload, ok := payload.RawPayload.(RadarrWebhookPayload)
	if !ok {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "invalid payload type for radarr handler",
		}, nil
	}

	switch payload.EventType {
	case EventDownload:
		return h.processDownload(&rawPayload)
	case EventGrab:
		return h.processGrab(&rawPayload)
	case EventRename:
		return h.processRename(&rawPayload)
	case EventTest:
		return &WebhookResult{
			Accepted:  true,
			MediaInfo: MediaInfo{Title: rawPayload.Movie.Title},
		}, nil
	default:
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "event type not supported for processing",
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

	return &WebhookResult{
		Accepted: true,
		Files: []File{
			{
				Path:         payload.MovieFile.Path,
				RelativePath: payload.MovieFile.RelativePath,
				Name:         payload.MovieFile.RelativePath,
				Size:         payload.MovieFile.Size,
			},
		},
		MediaInfo: MediaInfo{
			Title:    payload.Movie.Title,
			FilePath: payload.MovieFile.Path,
		},
	}, nil
}

func (h *RadarrHandler) processGrab(payload *RadarrWebhookPayload) (*WebhookResult, error) {
	var files []File
	for _, mf := range payload.MovieFiles {
		if mf.Path != "" {
			files = append(files, File{
				Path:         mf.Path,
				RelativePath: mf.RelativePath,
				Name:         mf.RelativePath,
				Size:         mf.Size,
			})
		}
	}

	if len(files) == 0 {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no movie files in grab event",
		}, nil
	}

	return &WebhookResult{
		Accepted:  true,
		Files:     files,
		MediaInfo: MediaInfo{Title: payload.Movie.Title},
	}, nil
}

func (h *RadarrHandler) processRename(payload *RadarrWebhookPayload) (*WebhookResult, error) {
	var files []File
	for _, rmf := range payload.RenamedMovieFiles {
		if rmf.Path != "" {
			files = append(files, File{
				Path:         rmf.Path,
				RelativePath: rmf.RelativePath,
				Name:         rmf.RelativePath,
			})
		}
	}

	if len(files) == 0 {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no renamed movie files in rename event",
		}, nil
	}

	return &WebhookResult{
		Accepted:  true,
		Files:     files,
		MediaInfo: MediaInfo{Title: payload.Movie.Title},
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

func (h *RadarrHandler) UnmarshalPayload(data []byte) (*RadarrWebhookPayload, error) {
	var payload RadarrWebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

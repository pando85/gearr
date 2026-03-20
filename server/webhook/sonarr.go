package webhook

import (
	"context"
	"encoding/json"
	"time"
)

type SonarrHandler struct {
	*BaseHandler
}

func NewSonarrHandler() *SonarrHandler {
	return &SonarrHandler{
		BaseHandler: NewBaseHandler(SourceSonarr, []EventType{
			EventDownload,
			EventGrab,
			EventRename,
			EventSeriesDelete,
			EventEpisodeFileDelete,
			EventTest,
		}),
	}
}

type SonarrWebhookPayload struct {
	EventType string `json:"eventType"`
	Series    struct {
		Title string `json:"title"`
		Path  string `json:"path"`
	} `json:"series"`
	EpisodeFile struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Size         int64  `json:"size"`
		Quality      string `json:"qualityVersion,omitempty"`
	} `json:"episodeFile"`
	EpisodeFiles []struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Size         int64  `json:"size"`
	} `json:"episodeFiles"`
	RenamedEpisodeFiles []struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"previousPath"`
	} `json:"renamedEpisodeFiles"`
}

func (h *SonarrHandler) Parse(ctx context.Context, payload []byte) (*WebhookPayload, error) {
	var rawPayload SonarrWebhookPayload
	if err := h.ParseRawPayload(payload, &rawPayload); err != nil {
		return nil, err
	}

	eventType := h.mapEventType(rawPayload.EventType)

	return &WebhookPayload{
		SourceType: SourceSonarr,
		EventType:  eventType,
		RawPayload: rawPayload,
		Timestamp:  time.Now(),
	}, nil
}

func (h *SonarrHandler) Process(ctx context.Context, payload *WebhookPayload) (*WebhookResult, error) {
	rawPayload, ok := payload.RawPayload.(SonarrWebhookPayload)
	if !ok {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "invalid payload type for sonarr handler",
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
			MediaInfo: MediaInfo{Title: rawPayload.Series.Title},
		}, nil
	default:
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "event type not supported for processing",
		}, nil
	}
}

func (h *SonarrHandler) processDownload(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	if payload.EpisodeFile.Path == "" {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no episode file path in download event",
		}, nil
	}

	return &WebhookResult{
		Accepted: true,
		Files: []File{
			{
				Path:         payload.EpisodeFile.Path,
				RelativePath: payload.EpisodeFile.RelativePath,
				Name:         payload.EpisodeFile.RelativePath,
				Size:         payload.EpisodeFile.Size,
			},
		},
		MediaInfo: MediaInfo{
			Title:    payload.Series.Title,
			FilePath: payload.EpisodeFile.Path,
		},
	}, nil
}

func (h *SonarrHandler) processGrab(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	var files []File
	for _, ef := range payload.EpisodeFiles {
		if ef.Path != "" {
			files = append(files, File{
				Path:         ef.Path,
				RelativePath: ef.RelativePath,
				Name:         ef.RelativePath,
				Size:         ef.Size,
			})
		}
	}

	if len(files) == 0 {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no episode files in grab event",
		}, nil
	}

	return &WebhookResult{
		Accepted:  true,
		Files:     files,
		MediaInfo: MediaInfo{Title: payload.Series.Title},
	}, nil
}

func (h *SonarrHandler) processRename(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	var files []File
	for _, ref := range payload.RenamedEpisodeFiles {
		if ref.Path != "" {
			files = append(files, File{
				Path:         ref.Path,
				RelativePath: ref.RelativePath,
				Name:         ref.RelativePath,
			})
		}
	}

	if len(files) == 0 {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no renamed episode files in rename event",
		}, nil
	}

	return &WebhookResult{
		Accepted:  true,
		Files:     files,
		MediaInfo: MediaInfo{Title: payload.Series.Title},
	}, nil
}

func (h *SonarrHandler) mapEventType(eventType string) EventType {
	switch eventType {
	case "Download":
		return EventDownload
	case "Grab":
		return EventGrab
	case "Rename":
		return EventRename
	case "SeriesDelete":
		return EventSeriesDelete
	case "EpisodeFileDelete":
		return EventEpisodeFileDelete
	case "Test":
		return EventTest
	default:
		return EventType(eventType)
	}
}

func (h *SonarrHandler) UnmarshalPayload(data []byte) (*SonarrWebhookPayload, error) {
	var payload SonarrWebhookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

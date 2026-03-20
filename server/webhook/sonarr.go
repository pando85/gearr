package webhook

import (
	"context"
	"time"

	"gearr/helper"
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
		Title  string `json:"title"`
		Path   string `json:"path"`
		TvdbId int    `json:"tvdbId"`
	} `json:"series"`
	Episodes []struct {
		Title          string `json:"title"`
		SeasonNumber   int    `json:"seasonNumber"`
		EpisodeNumber  int    `json:"episodeNumber"`
		QualityVersion int    `json:"qualityVersion"`
	} `json:"episodes"`
	EpisodeFile struct {
		RelativePath string `json:"relativePath"`
		Path         string `json:"path"`
		Size         int64  `json:"size"`
		Quality      string `json:"quality"`
	} `json:"episodeFile"`
	DeletedFiles []struct {
		Path string `json:"path"`
	} `json:"deletedFiles"`
	RenamedEpisodeFiles []struct {
		PreviousPath string `json:"previousPath"`
		Path         string `json:"path"`
		RelativePath string `json:"relativePath"`
	} `json:"renamedEpisodeFiles"`
	IsUpgrade bool `json:"isUpgrade"`
}

func (h *SonarrHandler) Parse(ctx context.Context, payload []byte) (*WebhookPayload, error) {
	var sonarrPayload SonarrWebhookPayload
	if err := h.ParseRawPayload(payload, &sonarrPayload); err != nil {
		return nil, err
	}

	eventType := h.mapEventType(sonarrPayload.EventType)

	return &WebhookPayload{
		SourceType: SourceSonarr,
		EventType:  eventType,
		RawPayload: sonarrPayload,
		Timestamp:  time.Now(),
	}, nil
}

func (h *SonarrHandler) Process(ctx context.Context, payload *WebhookPayload) (*WebhookResult, error) {
	sonarrPayload, ok := payload.RawPayload.(SonarrWebhookPayload)
	if !ok {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "invalid payload type",
		}, nil
	}

	switch payload.EventType {
	case EventDownload:
		return h.processDownload(&sonarrPayload)
	case EventGrab:
		return h.processGrab(&sonarrPayload)
	case EventRename:
		return h.processRename(&sonarrPayload)
	case EventSeriesDelete:
		return h.processSeriesDelete(&sonarrPayload)
	case EventEpisodeFileDelete:
		return h.processEpisodeFileDelete(&sonarrPayload)
	case EventTest:
		return h.processTest(&sonarrPayload)
	default:
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "unsupported event type",
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

	episodeTitle := ""
	if len(payload.Episodes) > 0 {
		episodeTitle = payload.Episodes[0].Title
	}

	helper.Infof("Sonarr download event: %s - %s", payload.Series.Title, payload.EpisodeFile.RelativePath)

	return &WebhookResult{
		Accepted: true,
		Files: []File{
			{
				Path:         payload.EpisodeFile.Path,
				RelativePath: payload.EpisodeFile.RelativePath,
				Name:         payload.EpisodeFile.RelativePath,
				Size:         payload.EpisodeFile.Size,
				Quality:      payload.EpisodeFile.Quality,
			},
		},
		MediaInfo: MediaInfo{
			Title:    payload.Series.Title + " - " + episodeTitle,
			FilePath: payload.EpisodeFile.Path,
		},
	}, nil
}

func (h *SonarrHandler) processGrab(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	helper.Debugf("Sonarr grab event: %s", payload.Series.Title)
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "grab event does not have completed download",
	}, nil
}

func (h *SonarrHandler) processRename(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	var files []File
	for _, renamed := range payload.RenamedEpisodeFiles {
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

	helper.Infof("Sonarr rename event: %d files renamed", len(files))

	return &WebhookResult{
		Accepted: true,
		Files:    files,
		MediaInfo: MediaInfo{
			Title:    payload.Series.Title,
			FilePath: payload.Series.Path,
		},
	}, nil
}

func (h *SonarrHandler) processSeriesDelete(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	helper.Infof("Sonarr series delete event: %s", payload.Series.Title)
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "series delete event - no action needed",
	}, nil
}

func (h *SonarrHandler) processEpisodeFileDelete(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	helper.Infof("Sonarr episode file delete event: %s", payload.Series.Title)
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "episode file delete event - no action needed",
	}, nil
}

func (h *SonarrHandler) processTest(payload *SonarrWebhookPayload) (*WebhookResult, error) {
	helper.Info("Sonarr test event received")
	return &WebhookResult{
		Accepted: true,
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

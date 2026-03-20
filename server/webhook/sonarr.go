package webhook

import (
	"context"
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
			EventTest,
			EventRename,
			EventEpisodeFileDelete,
			EventSeriesDelete,
		}),
	}
}

type SonarrSeries struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	TitleSlug        string   `json:"titleSlug"`
	Path             string   `json:"path"`
	TvdbID           int      `json:"tvdbId"`
	Overview         string   `json:"overview"`
	Year             int      `json:"year"`
	ImdbID           string   `json:"imdbId"`
	Genres           []string `json:"genres"`
	Network          string   `json:"network"`
	Status           string   `json:"status"`
	OriginalLanguage struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"originalLanguage"`
}

type SonarrEpisode struct {
	ID                       int       `json:"id"`
	EpisodeNumber            int       `json:"episodeNumber"`
	SeasonNumber             int       `json:"seasonNumber"`
	Title                    string    `json:"title"`
	AirDate                  string    `json:"airDate"`
	AirDateUtc               time.Time `json:"airDateUtc"`
	Overview                 string    `json:"overview"`
	HasFile                  bool      `json:"hasFile"`
	Monitored                bool      `json:"monitored"`
	AbsoluteEpisodeNumber    int       `json:"absoluteEpisodeNumber"`
	UnverifiedSceneNumbering bool      `json:"unverifiedSceneNumbering"`
}

type SonarrEpisodeFile struct {
	ID             int    `json:"id"`
	Path           string `json:"path"`
	RelativePath   string `json:"relativePath"`
	Size           int64  `json:"size"`
	Quality        string `json:"quality"`
	QualityVersion int    `json:"qualityVersion"`
	ReleaseGroup   string `json:"releaseGroup"`
	SceneName      string `json:"sceneName"`
}

type SonarrRelease struct {
	ReleaseTitle   string `json:"releaseTitle"`
	Indexer        string `json:"indexer"`
	ReleaseGroup   string `json:"releaseGroup"`
	Quality        string `json:"quality"`
	ReleaseType    string `json:"releaseType"`
	DownloadClient string `json:"downloadClient"`
	DownloadId     string `json:"downloadId"`
	Size           int64  `json:"size"`
}

type SonarrPayload struct {
	EventType    string            `json:"eventType"`
	InstanceName string            `json:"instanceName"`
	Application  string            `json:"application"`
	Series       SonarrSeries      `json:"series"`
	Episodes     []SonarrEpisode   `json:"episodes"`
	EpisodeFile  SonarrEpisodeFile `json:"episodeFile"`
	Release      SonarrRelease     `json:"release"`
	DeletedFiles []struct {
		Path string `json:"path"`
	} `json:"deletedFiles"`
}

func (h *SonarrHandler) Parse(ctx context.Context, payload []byte) (*WebhookPayload, error) {
	var sonarrPayload SonarrPayload
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

func (h *SonarrHandler) mapEventType(eventType string) EventType {
	switch eventType {
	case "Download":
		return EventDownload
	case "Grab":
		return EventGrab
	case "Test":
		return EventTest
	case "Rename":
		return EventRename
	case "EpisodeFileDelete":
		return EventEpisodeFileDelete
	case "SeriesDelete":
		return EventSeriesDelete
	default:
		return EventType(eventType)
	}
}

func (h *SonarrHandler) Process(ctx context.Context, payload *WebhookPayload) (*WebhookResult, error) {
	if payload.EventType == EventTest {
		return &WebhookResult{
			Accepted: true,
			MediaInfo: MediaInfo{
				Title: "Sonarr Test",
			},
		}, nil
	}

	sonarrPayload, ok := payload.RawPayload.(SonarrPayload)
	if !ok {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "invalid payload type",
		}, nil
	}

	switch payload.EventType {
	case EventDownload:
		return h.processDownload(sonarrPayload), nil
	case EventGrab:
		return h.processGrab(sonarrPayload), nil
	case EventRename:
		return h.processRename(sonarrPayload), nil
	case EventEpisodeFileDelete:
		return h.processEpisodeFileDelete(sonarrPayload), nil
	case EventSeriesDelete:
		return h.processSeriesDelete(sonarrPayload), nil
	default:
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "unsupported event type: " + string(payload.EventType),
		}, nil
	}
}

func (h *SonarrHandler) processDownload(payload SonarrPayload) *WebhookResult {
	if payload.EpisodeFile.Path == "" {
		return &WebhookResult{
			Accepted:   false,
			SkipReason: "no episode file path in download event",
		}
	}

	file := File{
		Path:         payload.EpisodeFile.Path,
		RelativePath: payload.EpisodeFile.RelativePath,
		Name:         extractFileName(payload.EpisodeFile.Path),
		Size:         payload.EpisodeFile.Size,
		Quality:      payload.EpisodeFile.Quality,
	}

	mediaInfo := MediaInfo{
		Title:    payload.Series.Title,
		FilePath: payload.EpisodeFile.Path,
	}

	return &WebhookResult{
		Accepted:  true,
		Files:     []File{file},
		MediaInfo: mediaInfo,
	}
}

func (h *SonarrHandler) processGrab(payload SonarrPayload) *WebhookResult {
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "grab events do not contain file paths",
	}
}

func (h *SonarrHandler) processRename(payload SonarrPayload) *WebhookResult {
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "rename events do not contain actionable file paths",
	}
}

func (h *SonarrHandler) processEpisodeFileDelete(payload SonarrPayload) *WebhookResult {
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "episode file delete events are informational only",
	}
}

func (h *SonarrHandler) processSeriesDelete(payload SonarrPayload) *WebhookResult {
	return &WebhookResult{
		Accepted:   false,
		SkipReason: "series delete events are informational only",
	}
}

func extractFileName(path string) string {
	if path == "" {
		return ""
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

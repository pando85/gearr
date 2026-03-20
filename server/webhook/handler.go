package webhook

import (
	"encoding/json"
	"gearr/helper"
	"gearr/model"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	config   model.WebhookConfig
	handlers map[model.WebhookSource]WebhookParser
}

type WebhookParser interface {
	Parse(body []byte) (*model.WebhookPayload, error)
	ValidateAuth(c *gin.Context, secret string) bool
}

func NewHandler(config model.WebhookConfig) *Handler {
	return &Handler{
		config:   config,
		handlers: make(map[model.WebhookSource]WebhookParser),
	}
}

func (h *Handler) RegisterParser(source model.WebhookSource, parser WebhookParser) {
	h.handlers[source] = parser
}

func (h *Handler) HandleRadarr(c *gin.Context) {
	if !h.config.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhooks are disabled"})
		return
	}

	parser, ok := h.handlers[model.WebhookSourceRadarr]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "radarr parser not registered"})
		return
	}

	if !parser.ValidateAuth(c, h.config.RadarrSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing authentication"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	payload, err := parser.Parse(body)
	if err != nil {
		helper.Errorf("failed to parse radarr webhook: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload.Source = model.WebhookSourceRadarr
	payload.RawBody = body

	c.JSON(http.StatusOK, gin.H{
		"status":     "received",
		"event_type": payload.EventType,
		"file_path":  payload.FilePath,
	})
}

func (h *Handler) HandleSonarr(c *gin.Context) {
	if !h.config.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhooks are disabled"})
		return
	}

	parser, ok := h.handlers[model.WebhookSourceSonarr]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sonarr parser not registered"})
		return
	}

	if !parser.ValidateAuth(c, h.config.SonarrSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing authentication"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	payload, err := parser.Parse(body)
	if err != nil {
		helper.Errorf("failed to parse sonarr webhook: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payload.Source = model.WebhookSourceSonarr
	payload.RawBody = body

	c.JSON(http.StatusOK, gin.H{
		"status":     "received",
		"event_type": payload.EventType,
		"file_path":  payload.FilePath,
	})
}

type radarrWebhook struct {
	EventType string `json:"eventType"`
	Movie     struct {
		FolderPath string `json:"folderPath"`
		Path       string `json:"path"`
	} `json:"movie"`
	MovieFile struct {
		Path string `json:"path"`
	} `json:"movieFile"`
}

type sonarrWebhook struct {
	EventType string `json:"eventType"`
	Series    struct {
		Path string `json:"path"`
	} `json:"series"`
	EpisodeFile struct {
		Path string `json:"path"`
	} `json:"episodeFile"`
}

type RadarrParser struct{}

func (p *RadarrParser) Parse(body []byte) (*model.WebhookPayload, error) {
	var raw radarrWebhook
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	payload := &model.WebhookPayload{
		EventType: model.WebhookEventType(raw.EventType),
	}

	switch payload.EventType {
	case model.WebhookEventDownload, model.WebhookEventGrab:
		if raw.MovieFile.Path != "" {
			payload.FilePath = raw.MovieFile.Path
		} else if raw.Movie.Path != "" {
			payload.FilePath = raw.Movie.Path
		}
	case model.WebhookEventRename, model.WebhookEventMovieAdded:
		if raw.Movie.FolderPath != "" {
			payload.MoviePath = raw.Movie.FolderPath
		}
	}

	return payload, nil
}

func (p *RadarrParser) ValidateAuth(c *gin.Context, secret string) bool {
	if secret == "" {
		return true
	}
	authHeader := c.GetHeader("X-Api-Key")
	if authHeader == "" {
		authHeader = c.GetHeader("Authorization")
	}
	return authHeader == secret
}

type SonarrParser struct{}

func (p *SonarrParser) Parse(body []byte) (*model.WebhookPayload, error) {
	var raw sonarrWebhook
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	payload := &model.WebhookPayload{
		EventType: model.WebhookEventType(raw.EventType),
	}

	switch payload.EventType {
	case model.WebhookEventDownload, model.WebhookEventGrab:
		if raw.EpisodeFile.Path != "" {
			payload.FilePath = raw.EpisodeFile.Path
		}
	case model.WebhookEventRename, model.WebhookEventSeriesAdd, model.WebhookEventEpisodeAdd:
		if raw.Series.Path != "" {
			payload.SeriesPath = raw.Series.Path
		}
	}

	return payload, nil
}

func (p *SonarrParser) ValidateAuth(c *gin.Context, secret string) bool {
	if secret == "" {
		return true
	}
	authHeader := c.GetHeader("X-Api-Key")
	if authHeader == "" {
		authHeader = c.GetHeader("Authorization")
	}
	return authHeader == secret
}

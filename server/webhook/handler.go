package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"gearr/helper"
	"gearr/model"

	"github.com/gin-gonic/gin"
)

type EventLogger interface {
	CreateWebhookEvent(ctx context.Context, event *model.WebhookEvent) error
}

type Handler interface {
	CanHandle(source Source, eventType EventType) bool
	Parse(ctx context.Context, payload []byte) (*WebhookPayload, error)
	Process(ctx context.Context, payload *WebhookPayload) (*WebhookResult, error)
	Source() Source
}

type BaseHandler struct {
	source     Source
	eventTypes []EventType
}

func NewBaseHandler(source Source, eventTypes []EventType) *BaseHandler {
	return &BaseHandler{
		source:     source,
		eventTypes: eventTypes,
	}
}

func (h *BaseHandler) CanHandle(source Source, eventType EventType) bool {
	if h.source != source {
		return false
	}
	for _, et := range h.eventTypes {
		if et == eventType {
			return true
		}
	}
	return false
}

func (h *BaseHandler) Source() Source {
	return h.source
}

func (h *BaseHandler) ParseRawPayload(payload []byte, target interface{}) error {
	return json.Unmarshal(payload, target)
}

type HandlerRegistry struct {
	handlers []Handler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make([]Handler, 0),
	}
}

func (r *HandlerRegistry) Register(handler Handler) {
	r.handlers = append(r.handlers, handler)
}

func (r *HandlerRegistry) GetHandler(source Source, eventType EventType) Handler {
	for _, h := range r.handlers {
		if h.CanHandle(source, eventType) {
			return h
		}
	}
	return nil
}

func (r *HandlerRegistry) Handlers() []Handler {
	return r.handlers
}

type HTTPHandler struct {
	registry    *HandlerRegistry
	eventLogger EventLogger
}

func NewHTTPHandler(registry *HandlerRegistry) *HTTPHandler {
	return &HTTPHandler{
		registry: registry,
	}
}

func NewHTTPHandlerWithLogger(registry *HandlerRegistry, logger EventLogger) *HTTPHandler {
	return &HTTPHandler{
		registry:    registry,
		eventLogger: logger,
	}
}

func NewDefaultHandlerRegistry() *HandlerRegistry {
	registry := NewHandlerRegistry()
	registry.Register(NewRadarrHandler())
	registry.Register(NewSonarrHandler())
	return registry
}

func (h *HTTPHandler) HandleWebhook(c *gin.Context) {
	sourceStr := c.Query("source")
	if sourceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source query parameter is required"})
		return
	}

	source := Source(sourceStr)
	eventTypeStr := c.Query("event")
	if eventTypeStr == "" {
		eventTypeStr = string(EventDownload)
	}
	eventType := EventType(eventTypeStr)

	handler := h.registry.GetHandler(source, eventType)
	if handler == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no handler found for source and event type"})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	payload, err := handler.Parse(c.Request.Context(), body)
	if err != nil {
		helper.Errorf("failed to parse webhook payload: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse webhook payload"})
		return
	}

	result, err := handler.Process(c.Request.Context(), payload)
	if err != nil {
		helper.Errorf("failed to process webhook: %v", err)
		h.logWebhookEvent(c.Request.Context(), source, eventType, result, nil, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

	h.logWebhookEvent(c.Request.Context(), source, eventType, result, body, nil)

	if !result.Accepted {
		c.JSON(http.StatusOK, gin.H{
			"accepted":    false,
			"skip_reason": result.SkipReason,
			"message":     "webhook received but not processed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accepted": true,
		"files":    result.Files,
		"message":  "webhook processed successfully",
	})
}

func (h *HTTPHandler) logWebhookEvent(ctx context.Context, source Source, eventType EventType, result *WebhookResult, rawPayload []byte, processErr error) {
	if h.eventLogger == nil {
		return
	}

	event := &model.WebhookEvent{
		Source:    model.WebhookProvider(source),
		EventType: string(eventType),
		CreatedAt: time.Now(),
	}

	if processErr != nil {
		event.Status = model.WebhookEventStatusFailed
		event.ErrorDetails = processErr.Error()
	} else if result != nil {
		if result.Accepted {
			event.Status = model.WebhookEventStatusSuccess
			if len(result.Files) > 0 {
				event.FilePath = result.Files[0].Path
			}
			event.Message = result.MediaInfo.Title
		} else {
			event.Status = model.WebhookEventStatusSkipped
			event.Message = result.SkipReason
		}
	}

	if rawPayload != nil {
		event.Payload = string(rawPayload)
	}

	if err := h.eventLogger.CreateWebhookEvent(ctx, event); err != nil {
		helper.Errorf("failed to log webhook event: %v", err)
	}
}

func (h *HTTPHandler) HandleTest(c *gin.Context) {
	sourceStr := c.Query("source")
	if sourceStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source query parameter is required"})
		return
	}

	source := Source(sourceStr)
	handler := h.registry.GetHandler(source, EventTest)
	if handler == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no test handler found for source"})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	payload, err := handler.Parse(c.Request.Context(), body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse test webhook payload"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accepted":    true,
		"source_type": payload.SourceType,
		"event_type":  payload.EventType,
		"message":     "test webhook received successfully",
	})
}

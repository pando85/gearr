package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	"gearr/helper"

	"github.com/gin-gonic/gin"
)

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
	registry *HandlerRegistry
}

func NewHTTPHandler(registry *HandlerRegistry) *HTTPHandler {
	return &HTTPHandler{
		registry: registry,
	}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

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

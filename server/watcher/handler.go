package watcher

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	watcher *Watcher
}

func NewHandler(watcher *Watcher) *Handler {
	return &Handler{watcher: watcher}
}

func (h *Handler) GetStatus(c *gin.Context) {
	if h.watcher == nil {
		c.JSON(http.StatusOK, WatcherStatus{Active: false})
		return
	}

	status := h.watcher.GetStatus()
	c.JSON(http.StatusOK, status)
}

func (h *Handler) GetDetections(c *gin.Context) {
	if h.watcher == nil {
		c.JSON(http.StatusOK, []*FileDetectionResponse{})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		limit = parseIntParam(l)
	}

	detections, err := h.watcher.GetRecentDetections(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detections)
}

func (h *Handler) AddPath(c *gin.Context) {
	if h.watcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "watcher not initialized"})
		return
	}

	var req AddPathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	if err := h.watcher.AddPath(req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "path added successfully"})
}

func (h *Handler) RemovePath(c *gin.Context) {
	if h.watcher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "watcher not initialized"})
		return
	}

	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query parameter is required"})
		return
	}

	if err := h.watcher.RemovePath(path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "path removed successfully"})
}

func (h *Handler) IsEnabled(c *gin.Context) {
	enabled := false
	if h.watcher != nil {
		enabled = h.watcher.IsEnabled()
	}
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

type AddPathRequest struct {
	Path string `json:"path"`
}

type FileDetectionResponse struct {
	Path       string `json:"path"`
	DetectedAt string `json:"detected_at"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

func parseIntParam(s string) int {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		result = result*10 + int(c-'0')
	}
	return result
}

package webhook

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSonarrHandler_Parse(t *testing.T) {
	handler := NewSonarrHandler()

	tests := []struct {
		name      string
		payload   string
		wantEvent EventType
		wantErr   bool
	}{
		{
			name: "download event",
			payload: `{
				"eventType": "Download",
				"series": {"title": "Test Series", "path": "/series/test"},
				"episodeFile": {"relativePath": "test.mkv", "path": "/series/test/test.mkv", "size": 1000000}
			}`,
			wantEvent: EventDownload,
			wantErr:   false,
		},
		{
			name: "grab event",
			payload: `{
				"eventType": "Grab",
				"series": {"title": "Test Series"},
				"episodeFiles": [{"relativePath": "test.mkv", "path": "/series/test/test.mkv", "size": 1000000}]
			}`,
			wantEvent: EventGrab,
			wantErr:   false,
		},
		{
			name: "test event",
			payload: `{
				"eventType": "Test",
				"series": {"title": "Test Series"}
			}`,
			wantEvent: EventTest,
			wantErr:   false,
		},
		{
			name:    "invalid json",
			payload: `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := handler.Parse(context.Background(), []byte(tt.payload))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && payload.EventType != tt.wantEvent {
				t.Errorf("Parse() eventType = %v, want %v", payload.EventType, tt.wantEvent)
			}
		})
	}
}

func TestSonarrHandler_Process(t *testing.T) {
	handler := NewSonarrHandler()

	tests := []struct {
		name         string
		payloadJSON  string
		wantAccepted bool
		wantFiles    int
	}{
		{
			name: "download with file",
			payloadJSON: `{
				"eventType": "Download",
				"series": {"title": "Test Series", "path": "/series/test"},
				"episodeFile": {"relativePath": "test.mkv", "path": "/series/test/test.mkv", "size": 1000000}
			}`,
			wantAccepted: true,
			wantFiles:    1,
		},
		{
			name: "download without file",
			payloadJSON: `{
				"eventType": "Download",
				"series": {"title": "Test Series"},
				"episodeFile": {}
			}`,
			wantAccepted: false,
		},
		{
			name: "test event",
			payloadJSON: `{
				"eventType": "Test",
				"series": {"title": "Test Series"}
			}`,
			wantAccepted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedPayload, err := handler.Parse(context.Background(), []byte(tt.payloadJSON))
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}
			result, err := handler.Process(context.Background(), parsedPayload)
			if err != nil {
				t.Errorf("Process() error = %v", err)
				return
			}
			if result.Accepted != tt.wantAccepted {
				t.Errorf("Process() accepted = %v, want %v", result.Accepted, tt.wantAccepted)
			}
			if tt.wantFiles > 0 && len(result.Files) != tt.wantFiles {
				t.Errorf("Process() files = %v, want %v", len(result.Files), tt.wantFiles)
			}
		})
	}
}

func TestRadarrHandler_Parse(t *testing.T) {
	handler := NewRadarrHandler()

	tests := []struct {
		name      string
		payload   string
		wantEvent EventType
		wantErr   bool
	}{
		{
			name: "download event",
			payload: `{
				"eventType": "Download",
				"movie": {"title": "Test Movie", "folderPath": "/movies/test"},
				"movieFile": {"relativePath": "test.mkv", "path": "/movies/test/test.mkv", "size": 1000000}
			}`,
			wantEvent: EventDownload,
			wantErr:   false,
		},
		{
			name: "test event",
			payload: `{
				"eventType": "Test",
				"movie": {"title": "Test Movie"}
			}`,
			wantEvent: EventTest,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := handler.Parse(context.Background(), []byte(tt.payload))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && payload.EventType != tt.wantEvent {
				t.Errorf("Parse() eventType = %v, want %v", payload.EventType, tt.wantEvent)
			}
		})
	}
}

func TestRadarrHandler_Process(t *testing.T) {
	handler := NewRadarrHandler()

	tests := []struct {
		name         string
		payloadJSON  string
		wantAccepted bool
		wantFiles    int
	}{
		{
			name: "download with file",
			payloadJSON: `{
				"eventType": "Download",
				"movie": {"title": "Test Movie", "folderPath": "/movies/test"},
				"movieFile": {"relativePath": "test.mkv", "path": "/movies/test/test.mkv", "size": 1000000}
			}`,
			wantAccepted: true,
			wantFiles:    1,
		},
		{
			name: "download without file",
			payloadJSON: `{
				"eventType": "Download",
				"movie": {"title": "Test Movie"},
				"movieFile": {}
			}`,
			wantAccepted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedPayload, err := handler.Parse(context.Background(), []byte(tt.payloadJSON))
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}
			result, err := handler.Process(context.Background(), parsedPayload)
			if err != nil {
				t.Errorf("Process() error = %v", err)
				return
			}
			if result.Accepted != tt.wantAccepted {
				t.Errorf("Process() accepted = %v, want %v", result.Accepted, tt.wantAccepted)
			}
		})
	}
}

func TestHandlerRegistry(t *testing.T) {
	registry := NewDefaultHandlerRegistry()

	radarrHandler := registry.GetHandler(SourceRadarr, EventDownload)
	if radarrHandler == nil {
		t.Error("Expected Radarr handler to be registered")
	}

	sonarrHandler := registry.GetHandler(SourceSonarr, EventDownload)
	if sonarrHandler == nil {
		t.Error("Expected Sonarr handler to be registered")
	}
}

func TestSonarrUnmarshalPayload(t *testing.T) {
	handler := NewSonarrHandler()
	jsonData := `{"eventType": "Download", "series": {"title": "Test"}}`

	payload, err := handler.UnmarshalPayload([]byte(jsonData))
	if err != nil {
		t.Errorf("UnmarshalPayload() error = %v", err)
		return
	}
	if payload.EventType != "Download" {
		t.Errorf("UnmarshalPayload() eventType = %v, want Download", payload.EventType)
	}
}

func TestRadarrUnmarshalPayload(t *testing.T) {
	handler := NewRadarrHandler()
	jsonData := `{"eventType": "Download", "movie": {"title": "Test"}}`

	payload, err := handler.UnmarshalPayload([]byte(jsonData))
	if err != nil {
		t.Errorf("UnmarshalPayload() error = %v", err)
		return
	}
	if payload.EventType != "Download" {
		t.Errorf("UnmarshalPayload() eventType = %v, want Download", payload.EventType)
	}
}

func TestBaseHandler_CanHandle(t *testing.T) {
	handler := NewBaseHandler(SourceRadarr, []EventType{EventDownload, EventGrab})

	if !handler.CanHandle(SourceRadarr, EventDownload) {
		t.Error("Expected handler to handle Radarr Download event")
	}
	if handler.CanHandle(SourceSonarr, EventDownload) {
		t.Error("Expected handler to not handle Sonarr Download event")
	}
	if handler.CanHandle(SourceRadarr, EventTest) {
		t.Error("Expected handler to not handle Radarr Test event")
	}
}

func TestWebhookPayload_JSONMarshal(t *testing.T) {
	payload := &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  EventDownload,
		RawPayload: map[string]interface{}{"test": "value"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Errorf("Failed to marshal payload: %v", err)
	}

	var unmarshaled WebhookPayload
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal payload: %v", err)
	}

	if unmarshaled.SourceType != SourceRadarr {
		t.Errorf("Source type mismatch: got %v, want %v", unmarshaled.SourceType, SourceRadarr)
	}
}

func TestWebhookResult_JSONMarshal(t *testing.T) {
	result := &WebhookResult{
		Accepted: true,
		Files: []File{
			{Path: "/test/file.mkv", RelativePath: "file.mkv", Name: "file.mkv", Size: 1000},
		},
		MediaInfo: MediaInfo{
			Title:    "Test Movie",
			FilePath: "/test/file.mkv",
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Errorf("Failed to marshal result: %v", err)
	}

	var unmarshaled WebhookResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal result: %v", err)
	}

	if !unmarshaled.Accepted {
		t.Error("Expected result to be accepted")
	}
	if len(unmarshaled.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(unmarshaled.Files))
	}
}

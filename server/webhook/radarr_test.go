package webhook

import (
	"context"
	"testing"
)

func TestRadarrHandler_CanHandle(t *testing.T) {
	handler := NewRadarrHandler()

	tests := []struct {
		name      string
		source    Source
		eventType EventType
		expected  bool
	}{
		{"radarr download", SourceRadarr, EventDownload, true},
		{"radarr grab", SourceRadarr, EventGrab, true},
		{"radarr test", SourceRadarr, EventTest, true},
		{"radarr unknown event", SourceRadarr, EventRename, false},
		{"sonarr download", SourceSonarr, EventDownload, false},
		{"unknown source", "unknown", EventDownload, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanHandle(tt.source, tt.eventType); got != tt.expected {
				t.Errorf("CanHandle() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRadarrHandler_Source(t *testing.T) {
	handler := NewRadarrHandler()
	if got := handler.Source(); got != SourceRadarr {
		t.Errorf("Source() = %v, want %v", got, SourceRadarr)
	}
}

func TestRadarrHandler_Parse_Download(t *testing.T) {
	handler := NewRadarrHandler()
	payload := `{
		"eventType": "Download",
		"movie": {
			"id": 1,
			"title": "Test Movie",
			"year": 2020,
			"filePath": "/movies/Test Movie (2020)/Test Movie.mkv",
			"folderPath": "/movies/Test Movie (2020)",
			"tmdbId": 12345,
			"imdbId": "tt1234567"
		},
		"movieFile": {
			"id": 1,
			"relativePath": "Test Movie.mkv",
			"path": "/movies/Test Movie (2020)/Test Movie.mkv",
			"quality": "Bluray-1080p",
			"size": 1234567890,
			"sceneName": "Test.Movie.2020.1080p.BluRay.x264-TEST"
		},
		"isUpgrade": false,
		"downloadId": "abc123"
	}`

	result, err := handler.Parse(context.Background(), []byte(payload))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.SourceType != SourceRadarr {
		t.Errorf("SourceType = %v, want %v", result.SourceType, SourceRadarr)
	}
	if result.EventType != EventDownload {
		t.Errorf("EventType = %v, want %v", result.EventType, EventDownload)
	}
}

func TestRadarrHandler_Parse_Grab(t *testing.T) {
	handler := NewRadarrHandler()
	payload := `{
		"eventType": "Grab",
		"movie": {
			"id": 1,
			"title": "Test Movie",
			"year": 2020
		},
		"release": {
			"indexer": "Test Indexer",
			"quality": "Bluray-1080p",
			"releaseTitle": "Test.Movie.2020.1080p.BluRay.x264-TEST",
			"size": 1234567890
		},
		"downloadId": "abc123"
	}`

	result, err := handler.Parse(context.Background(), []byte(payload))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.EventType != EventGrab {
		t.Errorf("EventType = %v, want %v", result.EventType, EventGrab)
	}
}

func TestRadarrHandler_Parse_Test(t *testing.T) {
	handler := NewRadarrHandler()
	payload := `{
		"eventType": "Test",
		"movie": {
			"id": 1,
			"title": "Test Movie",
			"year": 2020
		}
	}`

	result, err := handler.Parse(context.Background(), []byte(payload))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.EventType != EventTest {
		t.Errorf("EventType = %v, want %v", result.EventType, EventTest)
	}
}

func TestRadarrHandler_Process_Download(t *testing.T) {
	handler := NewRadarrHandler()
	payload := &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  EventDownload,
		RawPayload: radarrDownloadPayload{
			Movie: radarrMovie{
				Title:      "Test Movie",
				FolderPath: "/movies/Test Movie (2020)",
			},
			MovieFile: radarrMovieFile{
				Path:         "/movies/Test Movie (2020)/Test Movie.mkv",
				RelativePath: "Test Movie.mkv",
				SceneName:    "Test.Movie.2020.1080p.BluRay.x264-TEST",
				Size:         1234567890,
				Quality:      "Bluray-1080p",
			},
		},
	}

	result, err := handler.Process(context.Background(), payload)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if !result.Accepted {
		t.Errorf("Accepted = %v, want true", result.Accepted)
	}
	if len(result.Files) != 1 {
		t.Fatalf("Files count = %d, want 1", len(result.Files))
	}
	if result.Files[0].Path != "/movies/Test Movie (2020)/Test Movie.mkv" {
		t.Errorf("File path = %v, want /movies/Test Movie (2020)/Test Movie.mkv", result.Files[0].Path)
	}
	if result.MediaInfo.Title != "Test Movie" {
		t.Errorf("MediaInfo.Title = %v, want Test Movie", result.MediaInfo.Title)
	}
}

func TestRadarrHandler_Process_Grab(t *testing.T) {
	handler := NewRadarrHandler()
	payload := &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  EventGrab,
		RawPayload: radarrGrabPayload{
			Movie: radarrMovie{
				Title: "Test Movie",
			},
		},
	}

	result, err := handler.Process(context.Background(), payload)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.Accepted {
		t.Errorf("Accepted = %v, want false (grab events should not be processed)", result.Accepted)
	}
	if result.SkipReason == "" {
		t.Error("SkipReason should be set for grab events")
	}
}

func TestRadarrHandler_Process_Test(t *testing.T) {
	handler := NewRadarrHandler()
	payload := &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  EventTest,
	}

	result, err := handler.Process(context.Background(), payload)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if !result.Accepted {
		t.Errorf("Accepted = %v, want true", result.Accepted)
	}
}

func TestRadarrHandler_Process_Download_EmptyPath(t *testing.T) {
	handler := NewRadarrHandler()
	payload := &WebhookPayload{
		SourceType: SourceRadarr,
		EventType:  EventDownload,
		RawPayload: radarrDownloadPayload{
			Movie: radarrMovie{
				Title: "Test Movie",
			},
			MovieFile: radarrMovieFile{
				Path: "",
			},
		},
	}

	result, err := handler.Process(context.Background(), payload)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if result.Accepted {
		t.Errorf("Accepted = %v, want false (empty path should be rejected)", result.Accepted)
	}
	if result.SkipReason == "" {
		t.Error("SkipReason should be set when path is empty")
	}
}

func TestRadarrHandler_mapEventType(t *testing.T) {
	handler := NewRadarrHandler()

	tests := []struct {
		input    string
		expected EventType
	}{
		{"Download", EventDownload},
		{"Grab", EventGrab},
		{"Test", EventTest},
		{"MovieFileDelete", EventMovieDelete},
		{"MovieDelete", EventMovieDelete},
		{"Rename", EventRename},
		{"Unknown", EventDownload},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := handler.mapEventType(tt.input); got != tt.expected {
				t.Errorf("mapEventType(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

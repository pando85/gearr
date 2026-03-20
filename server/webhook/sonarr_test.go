package webhook

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewSonarrHandler(t *testing.T) {
	handler := NewSonarrHandler()

	if handler == nil {
		t.Fatal("NewSonarrHandler returned nil")
	}

	if handler.Source() != SourceSonarr {
		t.Errorf("Source() = %q, want %q", handler.Source(), SourceSonarr)
	}
}

func TestSonarrHandler_CanHandle(t *testing.T) {
	handler := NewSonarrHandler()

	tests := []struct {
		name      string
		source    Source
		eventType EventType
		want      bool
	}{
		{"download event", SourceSonarr, EventDownload, true},
		{"grab event", SourceSonarr, EventGrab, true},
		{"test event", SourceSonarr, EventTest, true},
		{"rename event", SourceSonarr, EventRename, true},
		{"episode file delete event", SourceSonarr, EventEpisodeFileDelete, true},
		{"series delete event", SourceSonarr, EventSeriesDelete, true},
		{"wrong source", SourceRadarr, EventDownload, false},
		{"unknown event", SourceSonarr, EventType("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.CanHandle(tt.source, tt.eventType); got != tt.want {
				t.Errorf("CanHandle(%q, %q) = %v, want %v", tt.source, tt.eventType, got, tt.want)
			}
		})
	}
}

func TestSonarrHandler_Parse(t *testing.T) {
	handler := NewSonarrHandler()
	ctx := context.Background()

	tests := []struct {
		name     string
		payload  SonarrPayload
		wantType EventType
		wantErr  bool
	}{
		{
			name: "download event",
			payload: SonarrPayload{
				EventType: "Download",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
					Path:  "/path/to/series",
				},
				EpisodeFile: SonarrEpisodeFile{
					ID:           1,
					Path:         "/path/to/series/S01E01.mkv",
					RelativePath: "S01E01.mkv",
					Size:         1000000000,
					Quality:      "HDTV-1080p",
				},
			},
			wantType: EventDownload,
			wantErr:  false,
		},
		{
			name: "grab event",
			payload: SonarrPayload{
				EventType: "Grab",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
				},
				Release: SonarrRelease{
					ReleaseTitle: "Test.Series.S01E01.1080p.WEB-DL",
					Indexer:      "test-indexer",
				},
			},
			wantType: EventGrab,
			wantErr:  false,
		},
		{
			name: "test event",
			payload: SonarrPayload{
				EventType:   "Test",
				Application: "Sonarr",
			},
			wantType: EventTest,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("failed to marshal test payload: %v", err)
			}

			result, err := handler.Parse(ctx, payloadBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result.EventType != tt.wantType {
				t.Errorf("Parse() EventType = %q, want %q", result.EventType, tt.wantType)
			}

			if result.SourceType != SourceSonarr {
				t.Errorf("Parse() SourceType = %q, want %q", result.SourceType, SourceSonarr)
			}
		})
	}
}

func TestSonarrHandler_Process(t *testing.T) {
	handler := NewSonarrHandler()
	ctx := context.Background()

	t.Run("test event", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventTest,
			RawPayload: SonarrPayload{
				EventType:   "Test",
				Application: "Sonarr",
			},
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if !result.Accepted {
			t.Error("Process() accepted = false, want true for test event")
		}

		if result.MediaInfo.Title != "Sonarr Test" {
			t.Errorf("Process() MediaInfo.Title = %q, want %q", result.MediaInfo.Title, "Sonarr Test")
		}
	})

	t.Run("download event with valid file", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventDownload,
			RawPayload: SonarrPayload{
				EventType: "Download",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
					Path:  "/path/to/series",
				},
				Episodes: []SonarrEpisode{
					{
						ID:            1,
						EpisodeNumber: 1,
						SeasonNumber:  1,
						Title:         "Test Episode",
					},
				},
				EpisodeFile: SonarrEpisodeFile{
					ID:           1,
					Path:         "/path/to/series/Season 01/S01E01.mkv",
					RelativePath: "Season 01/S01E01.mkv",
					Size:         1000000000,
					Quality:      "HDTV-1080p",
				},
			},
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if !result.Accepted {
			t.Error("Process() accepted = false, want true for download event")
		}

		if len(result.Files) != 1 {
			t.Errorf("Process() files count = %d, want 1", len(result.Files))
			return
		}

		if result.Files[0].Path != "/path/to/series/Season 01/S01E01.mkv" {
			t.Errorf("Process() file path = %q, want %q", result.Files[0].Path, "/path/to/series/Season 01/S01E01.mkv")
		}

		if result.Files[0].Quality != "HDTV-1080p" {
			t.Errorf("Process() quality = %q, want %q", result.Files[0].Quality, "HDTV-1080p")
		}

		if result.MediaInfo.Title != "Test Series" {
			t.Errorf("Process() MediaInfo.Title = %q, want %q", result.MediaInfo.Title, "Test Series")
		}
	})

	t.Run("download event without file path", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventDownload,
			RawPayload: SonarrPayload{
				EventType: "Download",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
				},
				EpisodeFile: SonarrEpisodeFile{
					Path: "",
				},
			},
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if result.Accepted {
			t.Error("Process() accepted = true, want false for download without file path")
		}

		if result.SkipReason != "no episode file path in download event" {
			t.Errorf("Process() skip reason = %q, want %q", result.SkipReason, "no episode file path in download event")
		}
	})

	t.Run("grab event", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventGrab,
			RawPayload: SonarrPayload{
				EventType: "Grab",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
				},
				Release: SonarrRelease{
					ReleaseTitle: "Test.Series.S01E01.1080p.WEB-DL",
				},
			},
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if result.Accepted {
			t.Error("Process() accepted = true, want false for grab event")
		}

		if result.SkipReason != "grab events do not contain file paths" {
			t.Errorf("Process() skip reason = %q, want %q", result.SkipReason, "grab events do not contain file paths")
		}
	})

	t.Run("rename event", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventRename,
			RawPayload: SonarrPayload{
				EventType: "Rename",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
				},
			},
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if result.Accepted {
			t.Error("Process() accepted = true, want false for rename event")
		}
	})

	t.Run("episode file delete event", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventEpisodeFileDelete,
			RawPayload: SonarrPayload{
				EventType: "EpisodeFileDelete",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
				},
			},
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if result.Accepted {
			t.Error("Process() accepted = true, want false for episode file delete event")
		}
	})

	t.Run("series delete event", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventSeriesDelete,
			RawPayload: SonarrPayload{
				EventType: "SeriesDelete",
				Series: SonarrSeries{
					ID:    1,
					Title: "Test Series",
				},
			},
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if result.Accepted {
			t.Error("Process() accepted = true, want false for series delete event")
		}
	})

	t.Run("invalid payload type", func(t *testing.T) {
		payload := &WebhookPayload{
			SourceType: SourceSonarr,
			EventType:  EventDownload,
			RawPayload: "invalid",
		}

		result, err := handler.Process(ctx, payload)
		if err != nil {
			t.Errorf("Process() error = %v", err)
			return
		}

		if result.Accepted {
			t.Error("Process() accepted = true, want false for invalid payload type")
		}

		if result.SkipReason != "invalid payload type" {
			t.Errorf("Process() skip reason = %q, want %q", result.SkipReason, "invalid payload type")
		}
	})
}

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"unix path", "/path/to/file.mkv", "file.mkv"},
		{"windows path", "C:\\path\\to\\file.mkv", "file.mkv"},
		{"filename only", "file.mkv", "file.mkv"},
		{"empty path", "", ""},
		{"trailing slash", "/path/to/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFileName(tt.path)
			if result != tt.expected {
				t.Errorf("extractFileName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestSonarrHandler_MapEventType(t *testing.T) {
	handler := NewSonarrHandler()

	tests := []struct {
		input    string
		expected EventType
	}{
		{"Download", EventDownload},
		{"Grab", EventGrab},
		{"Test", EventTest},
		{"Rename", EventRename},
		{"EpisodeFileDelete", EventEpisodeFileDelete},
		{"SeriesDelete", EventSeriesDelete},
		{"Unknown", EventType("Unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := handler.mapEventType(tt.input)
			if result != tt.expected {
				t.Errorf("mapEventType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

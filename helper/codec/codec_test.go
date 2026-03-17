package codec

import (
	"testing"
)

func TestNeedsTranscoding(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "x264 codec needs transcoding",
			path:     "/path/to/video.x264.mkv",
			expected: true,
		},
		{
			name:     "h264 codec needs transcoding",
			path:     "/path/to/video.h264.mkv",
			expected: true,
		},
		{
			name:     "H264 uppercase needs transcoding",
			path:     "/path/to/video.H264.mkv",
			expected: true,
		},
		{
			name:     "x265 codec does not need transcoding",
			path:     "/path/to/video.x265.mkv",
			expected: false,
		},
		{
			name:     "hevc codec does not need transcoding",
			path:     "/path/to/video.hevc.mkv",
			expected: false,
		},
		{
			name:     "mpeg-4 needs transcoding",
			path:     "/path/to/video.mpeg-4.mkv",
			expected: true,
		},
		{
			name:     "vp9 needs transcoding",
			path:     "/path/to/video.vp9.mkv",
			expected: true,
		},
		{
			name:     "av1 needs transcoding",
			path:     "/path/to/video.av1.mkv",
			expected: true,
		},
		{
			name:     "divx needs transcoding",
			path:     "/path/to/video.divx.mkv",
			expected: true,
		},
		{
			name:     "path without codec info",
			path:     "/path/to/video.mkv",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsTranscoding(tt.path)
			if result != tt.expected {
				t.Errorf("NeedsTranscoding(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFormatTargetName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "x264 to x265",
			path:     "/path/to/video.x264.mkv",
			expected: "/path/to/video.x265.mkv",
		},
		{
			name:     "h264 to x265",
			path:     "/path/to/video.h264.mp4",
			expected: "/path/to/video.x265.mkv",
		},
		{
			name:     "H264 uppercase to x265",
			path:     "/path/to/video.H264.avi",
			expected: "/path/to/video.x265.mkv",
		},
		{
			name:     "ac3 to AAC",
			path:     "/path/to/video.ac3.mkv",
			expected: "/path/to/video.AAC.mkv",
		},
		{
			name:     "no codec changes but extension to mkv",
			path:     "/path/to/video.mp4",
			expected: "/path/to/video.mkv",
		},
		{
			name:     "already mkv stays mkv",
			path:     "/path/to/video.mkv",
			expected: "/path/to/video.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTargetName(tt.path)
			if result != tt.expected {
				t.Errorf("FormatTargetName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

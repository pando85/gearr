package codec

import (
	"path/filepath"
	"regexp"
)

var (
	x264Regex = regexp.MustCompile(`(?i)(((x|h)264)|mpeg-4|mpeg-1|mpeg-2|mpeg|xvid|divx|vc-1|av1|vp8|vp9|wmv3|mp43)`)
	ac3Regex  = regexp.MustCompile(`(?i)(ac3|eac3|pcm|flac|mp2|dts|mp2|mp3|truehd|wma|vorbis|opus|mpeg audio)`)
)

func NeedsTranscoding(path string) bool {
	return x264Regex.MatchString(path)
}

func FormatTargetName(path string) string {
	p := x264Regex.ReplaceAllString(path, "x265")
	p = ac3Regex.ReplaceAllString(p, "AAC")
	extension := filepath.Ext(p)
	if extension != "" {
		p = p[:len(p)-len(extension)] + ".mkv"
	}
	return p
}

package task

import "testing"

func TestCalculateTesseractLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ger", "deu"},
		{"ge", "deu"},
		{"de", "deu"},
		{"en", "eng"},
		{"uk", "eng"},
		{"es", "spa"},
		{"esp", "spa"},
		{"fre", "fra"},
		{"chi", "chi_tra"},
		{"deu", "deu"},
		{"eng", "eng"},
		{"spa", "spa"},
		{"fra", "fra"},
		{"chi_tra", "chi_tra"},
		{"unknown", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := calculateTesseractLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("calculateTesseractLanguage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

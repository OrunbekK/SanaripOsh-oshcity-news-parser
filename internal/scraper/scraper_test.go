package scraper

import (
	"testing"
	"time"
)

func TestDateParserRussian(t *testing.T) {
	parser := NewDateParser("ru")

	tests := []struct {
		input    string
		expected time.Time
		wantErr  bool
	}{
		{"сегодня", time.Now().UTC().Truncate(24 * time.Hour), false},
		{"18 октября 2024", time.Date(2024, 10, 18, 0, 0, 0, 0, time.UTC), false},
		{"18 октября", time.Date(time.Now().Year(), 10, 18, 0, 0, 0, 0, time.UTC), false},
		{"18.10.2024", time.Date(2024, 10, 18, 0, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		result, err := parser.Parse(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if err == nil && !result.Equal(tt.expected) && tt.input != "сегодня" {
			t.Errorf("Parse(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/page#anchor", "https://example.com/page"},
		{"  https://example.com  ", "https://example.com"},
	}

	for _, tt := range tests {
		result := normalizeURL(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

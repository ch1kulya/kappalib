package views

import (
	"testing"
)

func TestResolveCover(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil url",
			input:    nil,
			expected: defaultCover,
		},
		{
			name:     "empty url",
			input:    ptr(""),
			expected: defaultCover,
		},
		{
			name:     "whitespace only",
			input:    ptr("   "),
			expected: defaultCover,
		},
		{
			name:     "valid http url",
			input:    ptr("http://example.com/cover.jpg"),
			expected: "http://example.com/cover.jpg",
		},
		{
			name:     "valid https url",
			input:    ptr("https://example.com/cover.jpg"),
			expected: "https://example.com/cover.jpg",
		},
		{
			name:     "valid https url with query and port",
			input:    ptr("https://example.com:8080/cover.jpg?v=1"),
			expected: "https://example.com:8080/cover.jpg?v=1",
		},
		{
			name:     "valid with leading trailing spaces",
			input:    ptr("  https://example.com/cover.jpg  "),
			expected: "https://example.com/cover.jpg",
		},
		{
			name:     "valid uppercase scheme",
			input:    ptr("HTTPS://example.com/cover.jpg"),
			expected: "HTTPS://example.com/cover.jpg",
		},
		{
			name:     "javascript scheme",
			input:    ptr("javascript:alert(1)"),
			expected: defaultCover,
		},
		{
			name:     "data scheme",
			input:    ptr("data:image/png;base64,abc"),
			expected: defaultCover,
		},
		{
			name:     "ftp scheme",
			input:    ptr("ftp://example.com/cover.jpg"),
			expected: defaultCover,
		},
		{
			name:     "relative scheme",
			input:    ptr("//example.com/cover.jpg"),
			expected: defaultCover,
		},
		{
			name:     "relative path",
			input:    ptr("/covers/1.jpg"),
			expected: defaultCover,
		},
		{
			name:     "no host http",
			input:    ptr("http://"),
			expected: defaultCover,
		},
		{
			name:     "no host https path",
			input:    ptr("https:///test.jpg"),
			expected: defaultCover,
		},
		{
			name:     "invalid url with control chars",
			input:    ptr("https://example.com/\n/test.jpg"),
			expected: defaultCover,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveCover(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveCover(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}

func TestIsExternalURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"http://example.com", true},
		{"https://example.com/page", true},
		{"/updates", false},
		{"catalog", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isExternalURL(tt.url); got != tt.expected {
			t.Errorf("isExternalURL(%q) = %v, want %v", tt.url, got, tt.expected)
		}
	}
}

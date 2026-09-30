package views

import (
	"os"
	"path/filepath"
	"strings"
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

func TestFormatCompactNumber(t *testing.T) {
	tests := []struct {
		n        int64
		expected string
	}{
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1_000, "1 тыс."},
		{1_049, "1 тыс."},
		{1_250, "1,2 тыс."},
		{9_999, "9,9 тыс."},
		{10_000, "10 тыс."},
		{15_780, "15 тыс."},
		{999_999, "999 тыс."},
		{1_000_000, "1 млн"},
		{1_234_567, "1,2 млн"},
		{12_345_678, "12 млн"},
		{999_999_999, "999 млн"},
		{1_500_000_000, "1,5 млрд"},
		{25_000_000_000, "25 млрд"},
	}

	for _, tt := range tests {
		if got := FormatCompactNumber(tt.n); got != tt.expected {
			t.Errorf("FormatCompactNumber(%d) = %q, want %q", tt.n, got, tt.expected)
		}
	}
}

func TestViewsLabel(t *testing.T) {
	tests := []struct {
		views    int64
		expected string
	}{
		{0, "0 просмотров"},
		{1, "1 просмотр"},
		{3, "3 просмотра"},
		{12, "12 просмотров"},
		{1_021, "1\u00a0021 просмотр"},
		{1_234_567, "1\u00a0234\u00a0567 просмотров"},
		{100_000_002, "100\u00a0000\u00a0002 просмотра"},
	}

	for _, tt := range tests {
		if got := viewsLabel(tt.views); got != tt.expected {
			t.Errorf("viewsLabel(%d) = %q, want %q", tt.views, got, tt.expected)
		}
	}
}

func TestViewsWord(t *testing.T) {
	tests := []struct {
		views    int64
		expected string
	}{
		{0, "просмотров"},
		{1, "просмотр"},
		{24, "просмотра"},
		{111, "просмотров"},
		{999, "просмотров"},
		{1_000, "просмотров"},
		{1_234_567, "просмотров"},
	}

	for _, tt := range tests {
		if got := viewsWord(tt.views); got != tt.expected {
			t.Errorf("viewsWord(%d) = %q, want %q", tt.views, got, tt.expected)
		}
	}
}

func TestCatalogRangeBadge(t *testing.T) {
	from, to := 2010, 2020
	tests := []struct {
		from     *int
		to       *int
		expected string
	}{
		{nil, nil, ""},
		{&from, nil, "от 2010"},
		{nil, &to, "до 2020"},
		{&from, &to, "2010–2020"},
	}

	for _, tt := range tests {
		if got := catalogRangeBadge(tt.from, tt.to); got != tt.expected {
			t.Errorf("catalogRangeBadge(%v, %v) = %q, want %q", tt.from, tt.to, got, tt.expected)
		}
	}
}

func TestBrandIconURL(t *testing.T) {
	tests := []struct {
		scheme   string
		expected string
	}{
		{"default", "/assets/icons/default/favicon.ico"},
		{"gruvbox", "/assets/icons/gruvbox/favicon.ico"},
		{"", "/assets/icons/default/favicon.ico"},
		{"../../etc", "/assets/icons/default/favicon.ico"},
	}

	for _, tt := range tests {
		if got := BrandIconURL(tt.scheme, "favicon.ico"); got != tt.expected {
			t.Errorf("BrandIconURL(%q) = %q, want %q", tt.scheme, got, tt.expected)
		}
	}
}

func TestBrandIconFilesExist(t *testing.T) {
	files := []string{"favicon.ico", "favicon-16x16.png", "favicon-32x32.png", "apple-touch-icon.png", "logo.png"}
	for _, s := range ColorSchemeOptions {
		for _, f := range files {
			path := filepath.Join("..", "..", "..", "assets", "static", strings.TrimPrefix(BrandIconURL(s.Value, f), "/assets/"))
			if _, err := os.Stat(path); err != nil {
				t.Errorf("missing brand icon %s for scheme %q: %v", f, s.Value, err)
			}
		}
	}
}

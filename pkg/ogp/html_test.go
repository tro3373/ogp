package ogp

import (
	"strings"
	"testing"
)

func TestExtractHTMLFallback_Title(t *testing.T) {
	htmlContent := `<html><head><title>Test Title</title></head><body></body></html>`
	fallback, err := ExtractHTMLFallback(strings.NewReader(htmlContent), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fallback.Title != "Test Title" {
		t.Errorf("got title %q, want %q", fallback.Title, "Test Title")
	}
}

func TestExtractHTMLFallback_Description(t *testing.T) {
	htmlContent := `<html><head><meta name="description" content="Test Description"></head></html>`
	fallback, err := ExtractHTMLFallback(strings.NewReader(htmlContent), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fallback.Description != "Test Description" {
		t.Errorf("got description %q, want %q", fallback.Description, "Test Description")
	}
}

func TestExtractHTMLFallback_Image(t *testing.T) {
	htmlContent := `<html><head><meta name="image" content="/img/logo.png"></head></html>`
	fallback, err := ExtractHTMLFallback(strings.NewReader(htmlContent), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://example.com/img/logo.png"
	if fallback.Image != want {
		t.Errorf("got image %q, want %q", fallback.Image, want)
	}
}

func TestExtractHTMLFallback_AllFields(t *testing.T) {
	htmlContent := `<html><head>
		<title>Full Page</title>
		<meta name="description" content="Full Description">
		<meta name="image" content="https://example.com/full.png">
	</head><body></body></html>`
	fallback, err := ExtractHTMLFallback(strings.NewReader(htmlContent), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fallback.Title != "Full Page" {
		t.Errorf("got title %q, want %q", fallback.Title, "Full Page")
	}
	if fallback.Description != "Full Description" {
		t.Errorf("got description %q, want %q", fallback.Description, "Full Description")
	}
	if fallback.Image != "https://example.com/full.png" {
		t.Errorf("got image %q, want %q", fallback.Image, "https://example.com/full.png")
	}
}

func TestResolveURL(t *testing.T) {
	tests := map[string]struct {
		baseURL string
		href    string
		want    string
	}{
		"absolute URL unchanged": {
			baseURL: "https://example.com",
			href:    "https://other.com/img.png",
			want:    "https://other.com/img.png",
		},
		"relative URL resolved": {
			baseURL: "https://example.com/page",
			href:    "/img/logo.png",
			want:    "https://example.com/img/logo.png",
		},
		"empty href returns empty": {
			baseURL: "https://example.com",
			href:    "",
			want:    "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := ResolveURL(tc.baseURL, tc.href)
			if got != tc.want {
				t.Errorf("ResolveURL(%q, %q) = %q, want %q", tc.baseURL, tc.href, got, tc.want)
			}
		})
	}
}

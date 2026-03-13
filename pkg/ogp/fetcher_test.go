package ogp

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type fakeHTTPClient struct {
	handler func(req *http.Request) ([]byte, int, error)
}

func (c *fakeHTTPClient) Request(req *http.Request) ([]byte, int, error) {
	return c.handler(req)
}

func TestFetch_GeneralURL_ReturnsOGPData(t *testing.T) {
	client := &fakeHTTPClient{
		handler: func(req *http.Request) ([]byte, int, error) {
			html := `<html><head>
				<meta property="og:title" content="Test Page">
				<meta property="og:description" content="Test Description">
				<meta property="og:image" content="https://example.com/img.png">
			</head><body></body></html>`
			return []byte(html), 200, nil
		},
	}
	fetcher := NewFetcher(client)
	result := fetcher.Fetch("https://example.com")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Title != "Test Page" {
		t.Errorf("got title %q, want %q", result.Title, "Test Page")
	}
	if result.Description != "Test Description" {
		t.Errorf("got description %q, want %q", result.Description, "Test Description")
	}
	if result.Image != "https://example.com/img.png" {
		t.Errorf("got image %q, want %q", result.Image, "https://example.com/img.png")
	}
}

func TestFetch_GeneralURL_HTMLFallback(t *testing.T) {
	client := &fakeHTTPClient{
		handler: func(req *http.Request) ([]byte, int, error) {
			html := `<html><head>
				<title>Fallback Title</title>
				<meta name="description" content="Fallback Description">
			</head><body><img src="/logo.png"></body></html>`
			return []byte(html), 200, nil
		},
	}
	fetcher := NewFetcher(client)
	result := fetcher.Fetch("https://example.com")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Title != "Fallback Title" {
		t.Errorf("got title %q, want %q", result.Title, "Fallback Title")
	}
	if result.Description != "Fallback Description" {
		t.Errorf("got description %q, want %q", result.Description, "Fallback Description")
	}
}

func TestFetch_TwitterURL_OEmbed(t *testing.T) {
	client := &fakeHTTPClient{
		handler: func(req *http.Request) ([]byte, int, error) {
			if strings.Contains(req.URL.String(), "publish.twitter.com/oembed") {
				body := `{
					"author_name": "TestUser",
					"html": "<blockquote class=\"twitter-tweet\"><p lang=\"en\" dir=\"ltr\">Hello from Twitter!</p></blockquote>"
				}`
				return []byte(body), 200, nil
			}
			return []byte("Not Found"), 404, nil
		},
	}
	fetcher := NewFetcher(client)
	result := fetcher.Fetch("https://x.com/TestUser/status/123")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Title != "@TestUser on X" {
		t.Errorf("got title %q, want %q", result.Title, "@TestUser on X")
	}
	if result.Description != "Hello from Twitter!" {
		t.Errorf("got description %q, want %q", result.Description, "Hello from Twitter!")
	}
}

func TestFetch_TwitterURL_OEmbedFallbackToGeneralOGP(t *testing.T) {
	client := &fakeHTTPClient{
		handler: func(req *http.Request) ([]byte, int, error) {
			if strings.Contains(req.URL.String(), "publish.twitter.com/oembed") {
				return []byte("Not Found"), 404, nil
			}
			html := `<html><head>
				<title>Twitter Page</title>
			</head></html>`
			return []byte(html), 200, nil
		},
	}
	fetcher := NewFetcher(client)
	result := fetcher.Fetch("https://x.com/user/status/456")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Title != "Twitter Page" {
		t.Errorf("got title %q, want %q", result.Title, "Twitter Page")
	}
}

func TestFetch_TwitterURL_TcoExpansion(t *testing.T) {
	client := &fakeHTTPClient{
		handler: func(req *http.Request) ([]byte, int, error) {
			url := req.URL.String()
			if strings.Contains(url, "publish.twitter.com/oembed") {
				body := `{
					"author_name": "LinkUser",
					"html": "<blockquote class=\"twitter-tweet\"><p lang=\"en\" dir=\"ltr\"><a href=\"https://t.co/abc\">https://t.co/abc</a></p></blockquote>"
				}`
				return []byte(body), 200, nil
			}
			if strings.Contains(url, "t.co/abc") {
				html := `<html><head>
					<meta property="og:title" content="Linked Article">
					<meta property="og:description" content="Article Description">
					<meta property="og:image" content="https://article.com/img.png">
				</head></html>`
				return []byte(html), 200, nil
			}
			return []byte("Not Found"), 404, nil
		},
	}
	fetcher := NewFetcher(client)
	result := fetcher.Fetch("https://x.com/LinkUser/status/789")

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Title != "@LinkUser on X" {
		t.Errorf("got title %q, want %q", result.Title, "@LinkUser on X")
	}
	if result.Description != "Linked Article" {
		t.Errorf("got description %q, want %q", result.Description, "Linked Article")
	}
	if result.Image != "https://article.com/img.png" {
		t.Errorf("got image %q, want %q", result.Image, "https://article.com/img.png")
	}
}

func TestFetch_HTTPError(t *testing.T) {
	client := &fakeHTTPClient{
		handler: func(req *http.Request) ([]byte, int, error) {
			return nil, 0, fmt.Errorf("connection refused")
		},
	}
	fetcher := NewFetcher(client)
	result := fetcher.Fetch("https://unreachable.example.com")

	if result.Err == nil {
		t.Error("expected error, got nil")
	}
}

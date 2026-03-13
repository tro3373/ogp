package ogp

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/dyatlov/go-opengraph/opengraph"
	log "github.com/sirupsen/logrus"
)

// HTTPClient is the interface for making HTTP requests and returning the response body.
type HTTPClient interface {
	Request(req *http.Request) (body []byte, statusCode int, err error)
}

// Fetcher fetches OGP metadata from URLs.
type Fetcher struct {
	client HTTPClient
}

// NewFetcher creates a new Fetcher with the given HTTP client.
func NewFetcher(client HTTPClient) *Fetcher {
	return &Fetcher{client: client}
}

// Fetch fetches OGP metadata from a URL.
func (f *Fetcher) Fetch(targetURL string) *Result {
	if IsTwitterURL(targetURL) {
		return f.fetchTwitter(targetURL)
	}
	return f.fetchGeneral(targetURL)
}

func (f *Fetcher) fetchGeneral(targetURL string) *Result {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return &Result{URL: targetURL, Err: fmt.Errorf("failed to create request for %s: %w", targetURL, err)}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ogp-cli/1.0)")

	body, statusCode, err := f.client.Request(req)
	if err != nil {
		return &Result{URL: targetURL, Err: fmt.Errorf("failed to fetch %s: %w", targetURL, err)}
	}
	if statusCode >= http.StatusBadRequest {
		return &Result{URL: targetURL, Err: fmt.Errorf("HTTP %d for %s", statusCode, targetURL)}
	}

	og := opengraph.NewOpenGraph()
	if err := og.ProcessHTML(bytes.NewReader(body)); err != nil {
		return &Result{URL: targetURL, Err: fmt.Errorf("failed to process HTML for %s: %w", targetURL, err)}
	}

	result := &Result{
		URL:         targetURL,
		Title:       og.Title,
		Description: og.Description,
	}
	if len(og.Images) > 0 {
		result.Image = og.Images[0].URL
	}

	applyFallback(result, body, targetURL)
	return result
}

func applyFallback(result *Result, htmlContent []byte, targetURL string) {
	if result.Title != "" && result.Description != "" && result.Image != "" {
		return
	}

	fallback, err := ExtractHTMLFallback(bytes.NewReader(htmlContent), targetURL)
	if err != nil {
		log.Warnf("failed to extract HTML fallback for %s: %v", targetURL, err)
		return
	}

	if result.Title == "" {
		result.Title = fallback.Title
	}
	if result.Description == "" {
		result.Description = fallback.Description
	}
	if result.Image == "" {
		result.Image = fallback.Image
	}
}

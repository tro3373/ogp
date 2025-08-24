package cmd

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/dyatlov/go-opengraph/opengraph/types/image"
	twitterscraper "github.com/imperatrona/twitter-scraper"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

func handleURL(url string) *TaskResult {
	if isTwitterURL(url) {
		return handleTwitterURL(url)
	}
	return handleGeneralURL(url)
}

func isTwitterURL(url string) bool {
	return strings.Contains(url, "twitter.com") || strings.Contains(url, "x.com")
}

func handleGeneralURL(url string) *TaskResult {
	// [http.Get に URI を変数のまま入れると叱られる](https://zenn.dev/spiegel/articles/20210125-http-get)
	resp, err := http.Get(url) //#nosec
	if err != nil {
		return &TaskResult{
			URL: url,
			Err: errors.Wrapf(err, "failed to handle url:%s", url),
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("failed to close response body for url", url, err)
		}
	}()

	// Read HTML content once and create multiple readers
	htmlContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TaskResult{
			URL: url,
			Err: errors.Wrapf(err, "failed to read response body for url:%s", url),
		}
	}

	// Try OpenGraph parsing first
	og := opengraph.NewOpenGraph()
	ogReader := strings.NewReader(string(htmlContent))
	if err := og.ProcessHTML(ogReader); err != nil {
		return &TaskResult{
			URL: url,
			Err: errors.Wrapf(err, "failed to ProcessHTML url:%s", url),
		}
	}

	// Check if OpenGraph data is incomplete and use HTML fallback
	title := og.Title
	description := og.Description
	image := ""
	if len(og.Images) > 0 {
		image = og.Images[0].URL
	}

	// If any crucial data is missing, try HTML fallback
	if title == "" || description == "" || image == "" {
		fallbackReader := strings.NewReader(string(htmlContent))
		fallback, err := extractHTMLFallback(fallbackReader, url)
		if err != nil {
			log.Warnf("Failed to extract HTML fallback for %s: %v", url, err)
		} else {
			if title == "" && fallback.Title != "" {
				title = fallback.Title
			}
			if description == "" && fallback.Description != "" {
				description = fallback.Description
			}
			if image == "" && fallback.Image != "" {
				image = fallback.Image
			}
		}
	}

	return &TaskResult{
		URL:         url,
		Title:       title,
		Description: description,
		Image:       image,
		Og:          og,
		Err:         nil,
	}
}

func handleTwitterURL(url string) *TaskResult {
	// Twitter/X.com専用の処理
	// xClientが設定されていればそれを使う
	if xClient != nil {
		return handleTwitterWithXClient(url)
	}
	// xClientがない場合は通常のOGP処理を使う
	return handleGeneralURL(url)
}

func handleTwitterWithXClient(url string) *TaskResult {
	// Extract tweet ID from URL
	tweetID := extractTweetID(url)
	if tweetID == "" {
		return &TaskResult{
			URL: url,
			Err: errors.New("failed to extract tweet ID from URL"),
		}
	}

	log.Debugf("Fetching tweet with ID: %s", tweetID)

	// Get tweet details using XClient
	tweet, err := xClient.Scraper.GetTweet(tweetID)
	if err != nil {
		return &TaskResult{
			URL: url,
			Err: errors.Wrapf(err, "failed to get tweet: %s", tweetID),
		}
	}

	// Create OpenGraph object from tweet data
	og := opengraph.NewOpenGraph()
	og.URL = url
	og.Title = "@" + tweet.Username + " on X"
	og.Description = tweet.Text
	og.Type = "article"
	og.SiteName = "X (formerly Twitter)"

	imageURL := getImageURL(tweet)
	if imageURL != "" {
		og.Images = append(og.Images, &image.Image{
			URL: imageURL,
		})
	}

	return &TaskResult{
		URL:         url,
		Title:       og.Title,
		Description: og.Description,
		Image:       imageURL,
		Og:          og,
		Err:         nil,
	}
}

func getImageURL(tweet *twitterscraper.Tweet) string {
	// Add images if present
	if len(tweet.Photos) > 0 {
		for _, photo := range tweet.Photos {
			return photo.URL
		}
	}

	// Add video preview if present
	if len(tweet.Videos) > 0 && tweet.Videos[0].Preview != "" {
		return tweet.Videos[0].Preview
	}

	tweetContent := tweet.Text
	url := getFirstURL(tweetContent)
	if url == "" {
		return ""
	}
	result := handleGeneralURL(url)
	return result.Image
}

func getFirstURL(text string) string {
	re := regexp.MustCompile(`https?://[^\s]+`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}

func extractTweetID(url string) string {
	// Handle various Twitter/X URL formats
	// https://twitter.com/username/status/1234567890
	// https://x.com/username/status/1234567890
	re := regexp.MustCompile(`(?:twitter\.com|x\.com)/[^/]+/status/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// HTMLFallbackData contains fallback metadata extracted from HTML
type HTMLFallbackData struct {
	Title       string
	Description string
	Image       string
}

// extractHTMLFallback extracts basic metadata from HTML as fallback
func extractHTMLFallback(reader io.Reader, baseURL string) (*HTMLFallbackData, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse HTML")
	}

	fallback := &HTMLFallbackData{}

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					fallback.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				name := getAttr(n, "name")
				content := getAttr(n, "content")

				if content != "" {
					switch name {
					case "description":
						fallback.Description = content
					case "image", "twitter:image":
						if fallback.Image == "" {
							fallback.Image = resolveURL(baseURL, content)
						}
					}
				}
			case "img":
				if fallback.Image == "" {
					src := getAttr(n, "src")
					if src != "" {
						fallback.Image = resolveURL(baseURL, src)
					}
				}
			case "link":
				if fallback.Image == "" {
					rel := getAttr(n, "rel")
					href := getAttr(n, "href")
					if href != "" && (rel == "icon" || rel == "shortcut icon" || rel == "apple-touch-icon") {
						fallback.Image = resolveURL(baseURL, href)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)
	return fallback, nil
}

// getAttr gets attribute value from HTML node
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// resolveURL converts relative URL to absolute URL
func resolveURL(baseURL, href string) string {
	if href == "" {
		return ""
	}

	// Parse base URL
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}

	// Parse href URL
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}

	// Resolve relative URL
	resolved := base.ResolveReference(ref)
	return resolved.String()
}

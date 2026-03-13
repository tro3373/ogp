package ogp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

const oEmbedAPIURL = "https://publish.twitter.com/oembed"

type oEmbedResponse struct {
	URL        string `json:"url"`
	AuthorName string `json:"author_name"`
	AuthorURL  string `json:"author_url"`
	HTML       string `json:"html"`
	Type       string `json:"type"`
}

// IsTwitterURL checks if the URL is a Twitter/X URL by matching the hostname.
func IsTwitterURL(targetURL string) bool {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	hosts := []string{
		"twitter.com",
		"www.twitter.com",
		"mobile.twitter.com",
		"x.com",
		"www.x.com",
		"mobile.x.com",
	}
	return slices.Contains(hosts, host)
}

func (f *Fetcher) fetchTwitter(tweetURL string) *Result {
	oembed, err := f.fetchOEmbed(tweetURL)
	if err != nil {
		log.Warnf("oEmbed API failed for %s: %v, falling back to general OGP", tweetURL, err)
		return f.fetchGeneral(tweetURL)
	}

	title := fmt.Sprintf("@%s on X", oembed.AuthorName)
	description := extractTextFromOEmbedHTML(oembed.HTML)
	image := ""

	linkedURLs := extractURLs(description)
	for _, u := range linkedURLs {
		if IsTwitterURL(u) {
			continue
		}
		linked := f.fetchLinkedContent(u)
		if linked == nil {
			continue
		}
		if isTcoOnly(description) && linked.Title != "" {
			description = linked.Title
		}
		if image == "" && linked.Image != "" {
			image = linked.Image
		}
		break
	}

	return &Result{
		URL:         tweetURL,
		Title:       title,
		Description: description,
		Image:       image,
	}
}

func (f *Fetcher) fetchOEmbed(tweetURL string) (*oEmbedResponse, error) {
	reqURL := fmt.Sprintf("%s?url=%s&omit_script=true", oEmbedAPIURL, url.QueryEscape(tweetURL))
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create oEmbed request: %w", err)
	}

	body, statusCode, err := f.client.Request(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch oEmbed: %w", err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("oEmbed API returned status %d", statusCode)
	}

	var oembed oEmbedResponse
	if err := json.Unmarshal(body, &oembed); err != nil {
		return nil, fmt.Errorf("failed to decode oEmbed response: %w", err)
	}

	return &oembed, nil
}

func (f *Fetcher) fetchLinkedContent(linkedURL string) *Result {
	linked := f.fetchGeneral(linkedURL)
	if linked.Err != nil {
		return nil
	}

	if linked.Title != "" {
		return linked
	}

	// Title is empty: the URL may have redirected to a Twitter page with no OGP
	// Try oEmbed for the original linked URL (e.g., t.co → x.com/i/article)
	redirectOembed, err := f.fetchOEmbed(linkedURL)
	if err != nil {
		return nil
	}

	return &Result{
		URL:         linkedURL,
		Title:       redirectOembed.AuthorName,
		Description: extractTextFromOEmbedHTML(redirectOembed.HTML),
		Image:       linked.Image,
	}
}

func extractTextFromOEmbedHTML(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return ""
	}

	text := findParagraphText(doc)
	return strings.TrimSpace(text)
}

func findParagraphText(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "p" {
		return extractNodeText(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text := findParagraphText(c)
		if text != "" {
			return text
		}
	}
	return ""
}

func extractNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractNodeText(c))
	}
	return sb.String()
}

var tcoPattern = regexp.MustCompile(`^\s*https?://t\.co/\S+\s*$`)
var urlPattern = regexp.MustCompile(`https?://[^\s]+`)

func isTcoOnly(text string) bool {
	return tcoPattern.MatchString(text)
}

func extractURLs(text string) []string {
	return urlPattern.FindAllString(text, -1)
}

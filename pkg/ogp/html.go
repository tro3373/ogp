package ogp

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// HTMLFallbackData contains fallback metadata extracted from HTML.
type HTMLFallbackData struct {
	Title       string
	Description string
	Image       string
}

// ExtractHTMLFallback extracts basic metadata from HTML as fallback.
func ExtractHTMLFallback(reader io.Reader, baseURL string) (*HTMLFallbackData, error) {
	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	fallback := &HTMLFallbackData{}
	traverseHTML(doc, fallback, baseURL)
	return fallback, nil
}

func traverseHTML(n *html.Node, fallback *HTMLFallbackData, baseURL string) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				fallback.Title = strings.TrimSpace(n.FirstChild.Data)
			}
		case "meta":
			handleMetaTag(n, fallback, baseURL)
		case "img":
			handleImgTag(n, fallback, baseURL)
		case "link":
			handleLinkTag(n, fallback, baseURL)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		traverseHTML(c, fallback, baseURL)
	}
}

func handleMetaTag(n *html.Node, fallback *HTMLFallbackData, baseURL string) {
	name := getAttr(n, "name")
	content := getAttr(n, "content")
	if content == "" {
		return
	}
	switch name {
	case "description":
		fallback.Description = content
	case "image", "twitter:image":
		if fallback.Image != "" {
			return
		}
		fallback.Image = ResolveURL(baseURL, content)
	}
}

func handleImgTag(n *html.Node, fallback *HTMLFallbackData, baseURL string) {
	if fallback.Image != "" {
		return
	}
	src := getAttr(n, "src")
	if src == "" {
		return
	}
	fallback.Image = ResolveURL(baseURL, src)
}

func handleLinkTag(n *html.Node, fallback *HTMLFallbackData, baseURL string) {
	if fallback.Image != "" {
		return
	}
	rel := getAttr(n, "rel")
	href := getAttr(n, "href")
	if href == "" {
		return
	}
	if rel != "icon" && rel != "shortcut icon" && rel != "apple-touch-icon" {
		return
	}
	fallback.Image = ResolveURL(baseURL, href)
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// ResolveURL converts a relative URL to an absolute URL.
func ResolveURL(baseURL, href string) string {
	if href == "" {
		return ""
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}

	ref, err := url.Parse(href)
	if err != nil {
		return href
	}

	return base.ResolveReference(ref).String()
}

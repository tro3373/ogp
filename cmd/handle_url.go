package cmd

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/dyatlov/go-opengraph/opengraph/types/image"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
	var reader io.Reader
	// [http.Get に URI を変数のまま入れると叱られる](https://zenn.dev/spiegel/articles/20210125-http-get)
	resp, err := http.Get(url) //#nosec
	if err != nil {
		return &TaskResult{
			URL: url,
			Err: errors.Wrapf(err, "failed to handle url:%s", url),
		}
	}
	reader = resp.Body
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error("failed to close response body for url", url, err)
		}
	}()
	og := opengraph.NewOpenGraph()
	if err := og.ProcessHTML(reader); err != nil {
		return &TaskResult{
			URL: url,
			Err: errors.Wrapf(err, "failed to ProcessHTML url:%s", url),
		}
	}
	image := ""
	if len(og.Images) > 0 {
		image = og.Images[0].URL // 最初の画像を使用
	}
	return &TaskResult{
		URL:         url,
		Title:       og.Title,
		Description: og.Description,
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

	// Add images if present
	if len(tweet.Photos) > 0 {
		for _, photo := range tweet.Photos {
			og.Images = append(og.Images, &image.Image{
				URL: photo.URL,
			})
		}
	}

	// Add video preview if present
	if len(tweet.Videos) > 0 && tweet.Videos[0].Preview != "" {
		og.Images = append(og.Images, &image.Image{
			URL: tweet.Videos[0].Preview,
		})
	}

	image := ""
	if len(og.Images) > 0 {
		image = og.Images[0].URL
	}

	return &TaskResult{
		URL:         url,
		Title:       og.Title,
		Description: og.Description,
		Image:       image,
		Og:          og,
		Err:         nil,
	}
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

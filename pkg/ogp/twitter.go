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
)

// defaultFxTwitterAPIBase is the public FixTweet instance. The unversioned
// /status/{id} route is used on purpose: the /2/ route returns the post
// under a "status" key with a different shape.
const defaultFxTwitterAPIBase = "https://api.fxtwitter.com"

// statusPathPattern matches the post path of an X URL and captures its numeric ID.
// The trailing boundary keeps /status/123abc from being read as post 123.
var statusPathPattern = regexp.MustCompile(`^/(?:i/web|[A-Za-z0-9_]{1,15})/status(?:es)?/(\d+)(?:[/?#]|$)`)

type fxTwitterResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Tweet   *fxTwitterStatus `json:"tweet"`
}

type fxTwitterStatus struct {
	Text   string `json:"text"`
	Author struct {
		ScreenName string `json:"screen_name"`
	} `json:"author"`
	Media struct {
		Photos []struct {
			URL string `json:"url"`
		} `json:"photos"`
		Videos []struct {
			ThumbnailURL string `json:"thumbnail_url"`
		} `json:"videos"`
	} `json:"media"`
	Card fxTwitterCard `json:"card"`
}

// fxTwitterCard is the link preview attached to a post. An absent card
// unmarshals to the zero value, so an empty Title is what "no card" looks like.
type fxTwitterCard struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Image struct {
		URL string `json:"url"`
	} `json:"image"`
}

// twitterHosts are the hostnames served by Twitter/X.
var twitterHosts = []string{
	"twitter.com",
	"www.twitter.com",
	"mobile.twitter.com",
	"x.com",
	"www.x.com",
	"mobile.x.com",
}

// parseTweetID returns the post ID of an X post URL, or "" for any other URL.
func parseTweetID(targetURL string) string {
	parsed, err := url.Parse(targetURL)
	if err != nil || !slices.Contains(twitterHosts, strings.ToLower(parsed.Hostname())) {
		return ""
	}
	matched := statusPathPattern.FindStringSubmatch(parsed.Path)
	if matched == nil {
		return ""
	}
	return matched[1]
}

func (f *Fetcher) fetchTwitter(tweetURL, tweetID string) *Result {
	tweet, err := f.fetchFxTwitter(tweetID)
	if err != nil {
		log.Warnf("fxtwitter API failed for %s: %v, falling back to general OGP", tweetURL, err)
		return f.fetchGeneral(tweetURL)
	}

	return &Result{
		URL:         tweetURL,
		Title:       fmt.Sprintf("@%s on X", tweet.Author.ScreenName),
		Description: tweetDescription(tweet),
		Image:       tweetImage(tweet),
	}
}

func (f *Fetcher) fetchFxTwitter(tweetID string) (*fxTwitterStatus, error) {
	reqURL, err := url.JoinPath(f.fxTwitterAPIBase, "status", tweetID)
	if err != nil {
		return nil, fmt.Errorf("failed to build fxtwitter request URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create fxtwitter request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	body, statusCode, err := f.client.Request(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fxtwitter: %w", err)
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("fxtwitter API returned status %d", statusCode)
	}

	var res fxTwitterResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to decode fxtwitter response: %w", err)
	}
	// A post with no author is unusable: it would surface as "@ on X" with no
	// error, so treat it the same as a missing post and let the caller fall back.
	if res.Tweet == nil || res.Tweet.Author.ScreenName == "" {
		return nil, fmt.Errorf("fxtwitter returned no usable tweet: %d %s", res.Code, res.Message)
	}

	return res.Tweet, nil
}

// resolveFxTwitterAPIBase validates the configured API base. A value that is
// not an absolute http(s) URL is rejected: joining it would silently produce a
// relative request URL and only fail once the request is sent.
func resolveFxTwitterAPIBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return defaultFxTwitterAPIBase
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		log.Warnf("invalid fxtwitter API base %q, falling back to %s", base, defaultFxTwitterAPIBase)
		return defaultFxTwitterAPIBase
	}
	return base
}

// tweetDescription prefers the post text, which already has t.co links expanded.
// A post whose text is nothing but the card link says nothing on its own,
// so the linked page title stands in for it.
func tweetDescription(tweet *fxTwitterStatus) string {
	text := strings.TrimSpace(tweet.Text)
	if tweet.Card.Title == "" {
		return text
	}
	if strings.TrimSpace(strings.ReplaceAll(text, tweet.Card.URL, "")) != "" {
		return text
	}
	return tweet.Card.Title
}

func tweetImage(tweet *fxTwitterStatus) string {
	for _, photo := range tweet.Media.Photos {
		if photo.URL != "" {
			return photo.URL
		}
	}
	for _, video := range tweet.Media.Videos {
		if video.ThumbnailURL != "" {
			return video.ThumbnailURL
		}
	}
	return tweet.Card.Image.URL
}

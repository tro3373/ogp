package ogp

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type fakeHTTPClient struct {
	handler  func(req *http.Request) ([]byte, int, error)
	requests []string
}

func (c *fakeHTTPClient) Request(req *http.Request) ([]byte, int, error) {
	c.requests = append(c.requests, req.URL.String())
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
	fetcher := NewFetcher(client, "")
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
	fetcher := NewFetcher(client, "")
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

func TestFetch_TwitterURL(t *testing.T) {
	const (
		generalHTML  = `<html><head><title>Fallback Page</title></head></html>`
		fxErrorHTML  = `<!DOCTYPE html><html><head></head><body></body></html>`
		notFoundJSON = `{"code":404,"message":"NOT_FOUND","tweet":null}`
		cardJSON     = `"card":{"url":"https://example.com/article","title":"An Article",` +
			`"image":{"url":"https://example.com/card.png"}}`
	)

	okJSON := func(tweet string) []byte {
		return []byte(`{"code":200,"message":"OK","tweet":{` + tweet + `}}`)
	}
	// fxtwitter へのリクエストだけ差し替え、それ以外は通常ページを返す
	fxHandler := func(fx func() ([]byte, int, error)) func(*http.Request) ([]byte, int, error) {
		return func(req *http.Request) ([]byte, int, error) {
			if strings.HasPrefix(req.URL.String(), defaultFxTwitterAPIBase) {
				return fx()
			}
			return []byte(generalHTML), 200, nil
		}
	}

	tests := map[string]struct {
		url             string
		apiBase         string
		handler         func(req *http.Request) ([]byte, int, error)
		wantTitle       string
		wantDescription string
		wantImage       string
		wantRequests    []string
	}{
		"投稿 URL の場合_fxtwitter の応答から OGP を組み立てること": {
			url: "https://x.com/jack/status/20",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(`"text":"just setting up my twttr","author":{"screen_name":"jack"}`), 200, nil
			}),
			wantTitle:       "@jack on X",
			wantDescription: "just setting up my twttr",
			wantRequests:    []string{"https://api.fxtwitter.com/status/20"},
		},
		"card 付き投稿の場合_本文と card の画像を返すこと": {
			url: "https://x.com/linker/status/21",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(`"text":"見て https://example.com/article",` +
					`"author":{"screen_name":"linker"},` + cardJSON), 200, nil
			}),
			wantTitle:       "@linker on X",
			wantDescription: "見て https://example.com/article",
			wantImage:       "https://example.com/card.png",
			wantRequests:    []string{"https://api.fxtwitter.com/status/21"},
		},
		"本文が card の URL だけの場合_説明が card の題名になること": {
			url: "https://x.com/linker/status/22",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(`"text":"https://example.com/article",` +
					`"author":{"screen_name":"linker"},` + cardJSON), 200, nil
			}),
			wantTitle:       "@linker on X",
			wantDescription: "An Article",
			wantImage:       "https://example.com/card.png",
			wantRequests:    []string{"https://api.fxtwitter.com/status/22"},
		},
		"写真付き投稿の場合_先頭写真が画像になること": {
			url: "https://x.com/shooter/status/23",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(`"text":"photo post","author":{"screen_name":"shooter"},` +
					`"media":{"photos":[{"type":"photo","url":"https://pbs.twimg.com/media/a.png"}]}`), 200, nil
			}),
			wantTitle:       "@shooter on X",
			wantDescription: "photo post",
			wantImage:       "https://pbs.twimg.com/media/a.png",
			wantRequests:    []string{"https://api.fxtwitter.com/status/23"},
		},
		"写真と動画と card が揃う場合_写真が画像になること": {
			url: "https://x.com/mixer/status/24",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(`"text":"mixed post","author":{"screen_name":"mixer"},` +
					`"media":{"photos":[{"url":"https://pbs.twimg.com/media/p.png"}],` +
					`"videos":[{"thumbnail_url":"https://pbs.twimg.com/media/v.jpg"}]},` + cardJSON), 200, nil
			}),
			wantTitle:       "@mixer on X",
			wantDescription: "mixed post",
			wantImage:       "https://pbs.twimg.com/media/p.png",
			wantRequests:    []string{"https://api.fxtwitter.com/status/24"},
		},
		"写真が無く動画と card がある場合_動画サムネイルが画像になること": {
			url: "https://x.com/vlogger/status/25",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(`"text":"video post","author":{"screen_name":"vlogger"},` +
					`"media":{"videos":[{"thumbnail_url":"https://pbs.twimg.com/media/v.jpg"}]},` +
					cardJSON), 200, nil
			}),
			wantTitle:       "@vlogger on X",
			wantDescription: "video post",
			wantImage:       "https://pbs.twimg.com/media/v.jpg",
			wantRequests:    []string{"https://api.fxtwitter.com/status/25"},
		},
		"fxtwitter が 404 の場合_通常の OGP 取得にフォールバックすること": {
			url: "https://x.com/jack/status/26",
			handler: fxHandler(func() ([]byte, int, error) {
				// ステータスで先に弾くのでボディは読まれない
				return nil, 404, nil
			}),
			wantTitle: "Fallback Page",
			wantRequests: []string{
				"https://api.fxtwitter.com/status/26",
				"https://x.com/jack/status/26",
			},
		},
		"tweet が null の場合_通常の OGP 取得にフォールバックすること": {
			url: "https://x.com/jack/status/27",
			handler: fxHandler(func() ([]byte, int, error) {
				return []byte(notFoundJSON), 200, nil
			}),
			wantTitle: "Fallback Page",
			wantRequests: []string{
				"https://api.fxtwitter.com/status/27",
				"https://x.com/jack/status/27",
			},
		},
		"author が空の場合_通常の OGP 取得にフォールバックすること": {
			url: "https://x.com/jack/status/28",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(``), 200, nil
			}),
			wantTitle: "Fallback Page",
			wantRequests: []string{
				"https://api.fxtwitter.com/status/28",
				"https://x.com/jack/status/28",
			},
		},
		"応答が JSON でない場合_通常の OGP 取得にフォールバックすること": {
			url: "https://x.com/jack/status/29",
			handler: fxHandler(func() ([]byte, int, error) {
				return []byte(fxErrorHTML), 200, nil
			}),
			wantTitle: "Fallback Page",
			wantRequests: []string{
				"https://api.fxtwitter.com/status/29",
				"https://x.com/jack/status/29",
			},
		},
		"投稿ページでない X URL の場合_通常の OGP 取得を使うこと": {
			url: "https://x.com/jack",
			handler: fxHandler(func() ([]byte, int, error) {
				return nil, 0, fmt.Errorf("fxtwitter should not be called")
			}),
			wantTitle:    "Fallback Page",
			wantRequests: []string{"https://x.com/jack"},
		},
		"API ベース URL が設定されている場合_その URL を叩くこと": {
			url:     "https://x.com/jack/status/20",
			apiBase: "https://fx.example.internal/",
			handler: func(req *http.Request) ([]byte, int, error) {
				return okJSON(`"text":"self hosted","author":{"screen_name":"jack"}`), 200, nil
			},
			wantTitle:       "@jack on X",
			wantDescription: "self hosted",
			wantRequests:    []string{"https://fx.example.internal/status/20"},
		},
		"API ベース URL が URL でない場合_既定の API を叩くこと": {
			url:     "https://x.com/jack/status/20",
			apiBase: "not-a-url",
			handler: fxHandler(func() ([]byte, int, error) {
				return okJSON(`"text":"just setting up my twttr","author":{"screen_name":"jack"}`), 200, nil
			}),
			wantTitle:       "@jack on X",
			wantDescription: "just setting up my twttr",
			wantRequests:    []string{"https://api.fxtwitter.com/status/20"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := &fakeHTTPClient{handler: tc.handler}
			result := NewFetcher(client, tc.apiBase).Fetch(tc.url)

			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}
			if result.Title != tc.wantTitle {
				t.Errorf("got title %q, want %q", result.Title, tc.wantTitle)
			}
			if result.Description != tc.wantDescription {
				t.Errorf("got description %q, want %q", result.Description, tc.wantDescription)
			}
			if result.Image != tc.wantImage {
				t.Errorf("got image %q, want %q", result.Image, tc.wantImage)
			}
			if !slices.Equal(client.requests, tc.wantRequests) {
				t.Errorf("got requests %q, want %q", client.requests, tc.wantRequests)
			}
		})
	}
}

func TestFetch_HTTPError(t *testing.T) {
	client := &fakeHTTPClient{
		handler: func(req *http.Request) ([]byte, int, error) {
			return nil, 0, fmt.Errorf("connection refused")
		},
	}
	fetcher := NewFetcher(client, "")
	result := fetcher.Fetch("https://unreachable.example.com")

	if result.Err == nil {
		t.Error("expected error, got nil")
	}
}

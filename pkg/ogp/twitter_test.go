package ogp

import (
	"testing"
)

func TestIsTwitterURL(t *testing.T) {
	tests := map[string]struct {
		url  string
		want bool
	}{
		"twitter.com URL": {
			url:  "https://twitter.com/user/status/123",
			want: true,
		},
		"x.com URL": {
			url:  "https://x.com/user/status/456",
			want: true,
		},
		"general URL": {
			url:  "https://example.com",
			want: false,
		},
		"github URL": {
			url:  "https://github.com/user/repo",
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := IsTwitterURL(tc.url)
			if got != tc.want {
				t.Errorf("IsTwitterURL(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}

func TestExtractTextFromOEmbedHTML(t *testing.T) {
	tests := map[string]struct {
		html string
		want string
	}{
		"standard tweet HTML": {
			html: `<blockquote class="twitter-tweet"><p lang="en" dir="ltr">Hello world!</p>— User (@user) <a href="https://twitter.com/user/status/123">March 1, 2024</a></blockquote>`,
			want: "Hello world!",
		},
		"tweet with link": {
			html: `<blockquote class="twitter-tweet"><p lang="en" dir="ltr">Check this: <a href="https://t.co/abc">https://t.co/abc</a></p></blockquote>`,
			want: "Check this: https://t.co/abc",
		},
		"empty HTML": {
			html: "",
			want: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := extractTextFromOEmbedHTML(tc.html)
			if got != tc.want {
				t.Errorf("extractTextFromOEmbedHTML() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsTcoOnly(t *testing.T) {
	tests := map[string]struct {
		text string
		want bool
	}{
		"t.co URL only": {
			text: "https://t.co/abc123",
			want: true,
		},
		"t.co with whitespace": {
			text: "  https://t.co/abc123  ",
			want: true,
		},
		"text with t.co": {
			text: "Check this https://t.co/abc123",
			want: false,
		},
		"no URL": {
			text: "Hello world",
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := isTcoOnly(tc.text)
			if got != tc.want {
				t.Errorf("isTcoOnly(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestExtractURLs(t *testing.T) {
	tests := map[string]struct {
		text string
		want int
	}{
		"single URL": {
			text: "Check https://example.com here",
			want: 1,
		},
		"multiple URLs": {
			text: "Visit https://a.com and http://b.com",
			want: 2,
		},
		"no URLs": {
			text: "No links here",
			want: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := extractURLs(tc.text)
			if len(got) != tc.want {
				t.Errorf("extractURLs(%q) returned %d URLs, want %d", tc.text, len(got), tc.want)
			}
		})
	}
}

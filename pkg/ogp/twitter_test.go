package ogp

import (
	"testing"
)

func TestParseTweetID(t *testing.T) {
	tests := map[string]struct {
		url  string
		want string
	}{
		"投稿 URL の場合_ID を返すこと": {
			url:  "https://x.com/jack/status/20",
			want: "20",
		},
		"statuses 形式の場合_ID を返すこと": {
			url:  "https://twitter.com/jack/statuses/20",
			want: "20",
		},
		"i/web 形式の場合_ID を返すこと": {
			url:  "https://x.com/i/web/status/20",
			want: "20",
		},
		"クエリ付きの場合_ID を返すこと": {
			url:  "https://x.com/jack/status/20?s=46&t=abc",
			want: "20",
		},
		"写真ページの場合_ID を返すこと": {
			url:  "https://x.com/jack/status/20/photo/1",
			want: "20",
		},
		"プロフィール URL の場合_空を返すこと": {
			url:  "https://x.com/jack",
			want: "",
		},
		"ID が数字でない場合_空を返すこと": {
			url:  "https://x.com/jack/status/abc",
			want: "",
		},
		"ID の末尾に数字以外が続く場合_空を返すこと": {
			url:  "https://x.com/jack/status/123abc",
			want: "",
		},
		"ID にアンダースコアが混ざる場合_空を返すこと": {
			url:  "https://x.com/jack/status/12_3",
			want: "",
		},
		"twitter.com の場合_ID を返すこと": {
			url:  "https://twitter.com/jack/status/20",
			want: "20",
		},
		"mobile.x.com の場合_ID を返すこと": {
			url:  "https://mobile.x.com/jack/status/20",
			want: "20",
		},
		"ホストが大文字の場合_ID を返すこと": {
			url:  "https://X.com/jack/status/20",
			want: "20",
		},
		"X 以外のホストの場合_空を返すこと": {
			url:  "https://example.com/jack/status/20",
			want: "",
		},
		"github URL の場合_空を返すこと": {
			url:  "https://github.com/user/repo",
			want: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseTweetID(tc.url)
			if got != tc.want {
				t.Errorf("parseTweetID(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}

package cmd

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"
)

// errReader fails partway through, like a broken pipe on stdin.
type errReader struct {
	prefix string
	read   bool
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, fmt.Errorf("broken pipe")
	}
	r.read = true
	return copy(p, r.prefix), nil
}

func TestReadUrls(t *testing.T) {
	tests := map[string]struct {
		reader  io.Reader
		want    []string
		wantErr bool
	}{
		"1 行 1 URL の場合_全て返すこと": {
			reader: strings.NewReader("https://a.example\nhttps://b.example\n"),
			want:   []string{"https://a.example", "https://b.example"},
		},
		"空行と余白がある場合_取り除くこと": {
			reader: strings.NewReader("  https://a.example  \n\n\t\nhttps://b.example\n"),
			want:   []string{"https://a.example", "https://b.example"},
		},
		"入力が空の場合_空を返すこと": {
			reader: strings.NewReader(""),
			want:   nil,
		},
		"読み取りに失敗した場合_エラーを返すこと": {
			reader:  &errReader{prefix: "https://a.example\n"},
			wantErr: true,
		},
		"1 行が長すぎる場合_エラーを返すこと": {
			reader:  strings.NewReader(strings.Repeat("a", bufio.MaxScanTokenSize+1)),
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := readUrls(tc.reader)
			if (err != nil) != tc.wantErr {
				t.Fatalf("readUrls() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("readUrls() = %q, want %q", got, tc.want)
			}
		})
	}
}

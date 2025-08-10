package cmd

import "testing"

func TestHandleUrl(t *testing.T) {
	url := "https://example.com/"
	tr := handleURL(url)
	if tr.Err != nil {
		t.Errorf("handleUrl(%s) returned error: %s", url, tr.Err)
		return
	}
	if tr.Og == nil {
		t.Errorf("handleUrl(%s) returned nil og", url)
		return
	}
	if tr.Og.Title != "" {
		t.Errorf("handleUrl(%s) returned wrong title: %s", url, tr.Og.Title)
	}
}

func TestHandleTwitterUrl(t *testing.T) {
	urls := []string{
		"https://twitter.com/user/status/123",
		"https://x.com/user/status/456",
	}

	for _, url := range urls {
		tr := handleURL(url)
		if !isTwitterURL(url) {
			t.Errorf("isTwitterURL(%s) should return true", url)
		}
		if tr.Err != nil {
			t.Errorf("handleUrl(%s) returned error: %s", url, tr.Err)
		}
	}
}

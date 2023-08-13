package cmd

import "testing"

func TestHandleUrl(t *testing.T) {
	url := "https://example.com/"
	ogp, err := handleUrl(url)
	if err != nil {
		t.Errorf("handleUrl(%s) returned error: %s", url, err)
	}
	if ogp == nil {
		t.Errorf("handleUrl(%s) returned nil", url)
	}
	if ogp.Title != "" {
		t.Errorf("handleUrl(%s) returned wrong title: %s", url, ogp.Title)
	}
}

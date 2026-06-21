package dev_tools

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestDanbooruTags(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}
	url := "https://danbooru.donmai.us/tags.json?search[name_matches]=*jeanne*&search[order]=count&limit=10"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "matoi/1.0.0")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Status: %d\nBody: %s\n", resp.StatusCode, string(body))
}

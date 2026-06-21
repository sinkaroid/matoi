package dev_tools_test

import (
	"crypto/tls"
	"io"
	"net/http"
	"testing"
)

func TestRule34Fetch(t *testing.T) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", "https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=2", nil)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Matoi/1.0")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Status: %d", resp.StatusCode)
	t.Logf("Body length: %d", len(body))
	if len(body) > 500 {
		t.Logf("Body: %s", string(body[:500]))
	} else {
		t.Logf("Body: %s", string(body))
	}
}

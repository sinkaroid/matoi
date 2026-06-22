package dev_tools_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

type FurbooruImage struct {
	ID              int               `json:"id"`
	Tags            []string          `json:"tags"`
	SourceURL       string            `json:"source_url"`
	Representations map[string]string `json:"representations"`
}

type FurbooruResponse struct {
	Images []FurbooruImage `json:"images"`
}

func TestFurbooruPrototype(t *testing.T) {
	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		t.Logf("No .env file found or failed to load: %v", err)
	}

	baseURL := os.Getenv("FURBOORU_URL")
	if baseURL == "" {
		baseURL = "https://furbooru.org/api/v1"
	}
	// Append the search endpoint
	endpointURL := baseURL + "/json/search/images"

	apiKey := os.Getenv("FURBOORU_API_KEY")

	params := url.Values{}
	params.Add("q", "bikini")
	params.Add("per_page", "5")
	params.Add("page", "1")
	if apiKey != "" {
		params.Add("key", apiKey)
	}

	requestURL := fmt.Sprintf("%s?%s", endpointURL, params.Encode())
	t.Logf("Fetching: %s", requestURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Matoi user-agent
	req.Header.Set("User-Agent", "Matoi_Project/1.0")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	// Print raw JSON output for the user to see
	t.Logf("Raw JSON Output:\n%s\n", string(body))

	var data FurbooruResponse
	if err := json.Unmarshal(body, &data); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v\nBody: %s", err, string(body))
	}

	t.Logf("Successfully fetched %d posts.", len(data.Images))

	for i, img := range data.Images {
		t.Logf("[%d] ID: %d | Full URL: %s", i+1, img.ID, img.Representations["full"])
	}

	if len(data.Images) == 0 {
		t.Errorf("Expected to get results but got 0")
	}
}

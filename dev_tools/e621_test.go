package dev_tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestE621QuickQuery(t *testing.T) {
	_ = godotenv.Load("../.env")

	username := os.Getenv("E621_API_ID")
	apiKey := os.Getenv("E621_API_KEY")
	baseURL := os.Getenv("E621_URL")
	if baseURL == "" {
		baseURL = "https://e621.net/posts.json"
	}
	limit := os.Getenv("E621_RETURN_LIMIT")
	if limit == "" {
		limit = "5"
	}

	url := fmt.Sprintf("%s?tags=yuri&limit=%s&page=1", baseURL, limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.SetBasicAuth(username, apiKey)
	req.Header.Set("User-Agent", "Matoi_Project/1.0 (by "+username+" on e621)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	posts, ok := result["posts"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'posts' array in response")
	}

	fmt.Printf("Successfully fetched %d posts from e621\n", len(posts))
	if len(posts) > 0 {
		firstPost, _ := posts[0].(map[string]interface{})
		fmt.Printf("First Post ID: %v\n", firstPost["id"])

		prettyJSON, _ := json.MarshalIndent(firstPost, "", "  ")
		fmt.Printf("First Post JSON:\n%s\n", string(prettyJSON))
	}
}

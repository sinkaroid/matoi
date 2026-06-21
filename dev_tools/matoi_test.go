package dev_tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

const baseURL = "http://localhost:3000"
const apiKey = "matoi"

var providers = []string{"rule34", "danbooru", "gelbooru", "tbib", "xbooru"}

// MatoiPost minimal struct to extract matoi_file_url
type MatoiPost struct {
	MatoiFileURL string `json:"matoi_file_url"`
}

type PostsResponse struct {
	Success bool        `json:"success"`
	Posts   []MatoiPost `json:"posts"`
}

func doAuthRequest(t *testing.T, method, url string, body io.Reader) *http.Response {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	return resp
}

func TestMatoiAllProviders(t *testing.T) {
	for _, p := range providers {
		t.Run(p, func(t *testing.T) {
			// 1. Post Test
			postURL := fmt.Sprintf("%s/api/%s/posts?tags=yuri&page=1&limit=3", baseURL, p)
			resp := doAuthRequest(t, "GET", postURL, nil)
			if resp.StatusCode != 200 {
				t.Fatalf("[%s] Post test failed with status %d", p, resp.StatusCode)
			}
			
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var postsData PostsResponse
			if err := json.Unmarshal(body, &postsData); err != nil {
				t.Fatalf("[%s] Failed to parse JSON: %v", p, err)
			}
			
			t.Logf("[%s] Get Posts successful. Fetched %d posts.", p, len(postsData.Posts))

			if len(postsData.Posts) == 0 {
				t.Logf("[%s] Warning: No posts found for tags=yuri", p)
			} else {
				// 2. Media Test
				mediaURL := postsData.Posts[0].MatoiFileURL
				if mediaURL == "" {
					t.Fatalf("[%s] matoi_file_url is empty in response", p)
				}
				
				// Some Matoi proxy URLs might not require auth, but we send it just in case
				mediaResp := doAuthRequest(t, "GET", mediaURL, nil)
				defer mediaResp.Body.Close()
				if mediaResp.StatusCode != 200 {
					t.Fatalf("[%s] Media proxy test failed with status %d for URL: %s", p, mediaResp.StatusCode, mediaURL)
				}
				t.Logf("[%s] Media proxy successful for Matoi File URL: %s", p, mediaURL)
			}

			// 3. Query Completion Test
			qcURL := fmt.Sprintf("%s/api/%s/query_completion?tags=jeanne", baseURL, p)
			qcResp := doAuthRequest(t, "GET", qcURL, nil)
			defer qcResp.Body.Close()
			if qcResp.StatusCode != 200 {
				t.Fatalf("[%s] Query completion test failed with status %d", p, qcResp.StatusCode)
			}
			
			qcBody, _ := io.ReadAll(qcResp.Body)
			var qcData map[string]interface{}
			_ = json.Unmarshal(qcBody, &qcData)
			
			tags, _ := qcData["tags"].([]interface{})
			displayCount := 3
			if len(tags) < displayCount {
				displayCount = len(tags)
			}
			t.Logf("[%s] Query completion successful. First %d tags: %v", p, displayCount, tags[:displayCount])
		})
	}
}

func TestMatoiGraphQLAllProviders(t *testing.T) {
	// 4. GraphQL Test (3 3 nya lewat graphql)
	// We will query posts and completion for each provider.
	queryStr := "query {"
	for _, p := range providers {
		queryStr += fmt.Sprintf(`
			%s {
				posts(tags: "yuri", limit: 3, page: 1) {
					id
					matoi_file_url
				}
				completion(tags: "jeanne")
			}
		`, p)
	}
	queryStr += "}"

	payload := map[string]string{
		"query": queryStr,
	}
	
	jsonPayload, _ := json.Marshal(payload)
	resp := doAuthRequest(t, "POST", baseURL+"/api/graphql", bytes.NewBuffer(jsonPayload))
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("GraphQL query failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse GraphQL JSON response: %v", err)
	}
	
	if errs, hasErrors := result["errors"]; hasErrors {
		t.Fatalf("GraphQL returned errors: %v", errs)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("GraphQL response missing data field")
	}

	for _, p := range providers {
		t.Run("GraphQL_"+p, func(t *testing.T) {
			providerData, ok := data[p].(map[string]interface{})
			if !ok {
				t.Fatalf("Missing data for provider: %s", p)
			}
			
			// Verify completion exists
			completions, _ := providerData["completion"].([]interface{})
			if len(completions) == 0 {
				t.Logf("[%s] Warning: No completions found for tags=jeanne", p)
			} else {
				displayCount := 3
				if len(completions) < displayCount {
					displayCount = len(completions)
				}
				t.Logf("[%s] GraphQL Query Completion successful. First %d tags: %v", p, displayCount, completions[:displayCount])
			}

			// Verify posts and media proxy
			posts, ok := providerData["posts"].([]interface{})
			if !ok || len(posts) == 0 {
				t.Logf("[%s] Warning: No posts found for tags=yuri", p)
				return
			}
			t.Logf("[%s] GraphQL Get Posts successful. Fetched %d posts.", p, len(posts))

			firstPost := posts[0].(map[string]interface{})
			matoiFileURL, ok := firstPost["matoi_file_url"].(string)
			if !ok || matoiFileURL == "" {
				t.Fatalf("[%s] matoi_file_url is empty in GraphQL response", p)
			}

			// Media Test using URL from GraphQL
			mediaResp := doAuthRequest(t, "GET", matoiFileURL, nil)
			defer mediaResp.Body.Close()
			if mediaResp.StatusCode != 200 {
				t.Fatalf("[%s] GraphQL Media proxy test failed with status %d for URL: %s", p, mediaResp.StatusCode, matoiFileURL)
			}
			t.Logf("[%s] GraphQL Media proxy successful for Matoi File URL: %s", p, matoiFileURL)
		})
	}

	t.Log("GraphQL test (Posts, Completion, Media) successful for all providers")
}

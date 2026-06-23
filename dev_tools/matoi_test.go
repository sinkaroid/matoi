package dev_tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

const (
	baseURL = "http://localhost:3000"
	apiKey  = "matoi"
)

var providers = []string{"rule34", "danbooru", "gelbooru", "tbib", "xbooru", "hypnohub", "safebooru", "yandere", "konachan_com", "konachan_net", "e621", "e926", "furbooru", "derpibooru"}

// MatoiPost minimal struct to extract matoi_file_url
type MatoiPost struct {
	ID           int    `json:"id"`
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
			var postsData PostsResponse
			var activeTag string
			for _, tag := range []string{"yuri", "1girl", "bikini"} {
				postURL := fmt.Sprintf("%s/api/%s/posts?tags=%s&page=1&limit=3", baseURL, p, tag)
				resp := doAuthRequest(t, "GET", postURL, nil)
				if resp.StatusCode == 200 {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err := json.Unmarshal(body, &postsData); err == nil && len(postsData.Posts) > 0 {
						activeTag = tag
						break
					}
				} else if resp.StatusCode != 404 {
					if p == "konachan_com" && resp.StatusCode == 500 {
						t.Logf("[%s] Warning: Post test returned 500, likely due to Cloudflare 403. Skipping.", p)
						activeTag = tag
						break
					}
					t.Fatalf("[%s] Post test failed with status %d", p, resp.StatusCode)
				}
				resp.Body.Close()
			}

			if activeTag == "" && p != "konachan_com" {
				t.Fatalf("[%s] No posts found for tags yuri, 1girl, or bikini", p)
			}

			if p == "konachan_com" && len(postsData.Posts) == 0 {
				t.Logf("[%s] Skipping remaining tests due to CF block.", p)
				return
			}

			t.Logf("[%s] Get Posts (Page 1) successful with tag '%s'. Fetched %d posts.", p, activeTag, len(postsData.Posts))

			// Verify pagination by fetching Page 2
			postURL2 := fmt.Sprintf("%s/api/%s/posts?tags=%s&page=2&limit=3", baseURL, p, activeTag)
			resp2 := doAuthRequest(t, "GET", postURL2, nil)
			if resp2.StatusCode == 200 {
				body2, _ := io.ReadAll(resp2.Body)
				resp2.Body.Close()
				var postsData2 PostsResponse
				if err := json.Unmarshal(body2, &postsData2); err == nil && len(postsData2.Posts) > 0 && len(postsData.Posts) > 0 {
					idPage1 := postsData.Posts[0].ID
					idPage2 := postsData2.Posts[0].ID
					t.Logf("[%s] Pagination Verified -> Page 1 First Post ID: %v | Page 2 First Post ID: %v", p, idPage1, idPage2)
					if idPage1 == idPage2 {
						t.Fatalf("[%s] Pagination failed! Page 1 and Page 2 returned the exact same Post ID: %v", p, idPage1)
					}
				}
			}

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
				p1: posts(tags: "yuri", limit: 3, page: 1) {
					id
					matoi_file_url
				}
				p2: posts(tags: "1girl", limit: 3, page: 1) {
					id
					matoi_file_url
				}
				p3: posts(tags: "bikini", limit: 3, page: 1) {
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
		// GraphQL returns errors if konachan_com fails due to Cloudflare when flaresolverr is off.
		// We shouldn't fail the whole test suite, just log it.
		t.Logf("GraphQL returned errors (expected if konachan_com is blocked): %v", errs)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("GraphQL response missing data field")
	}

	for _, p := range providers {
		t.Run("GraphQL_"+p, func(t *testing.T) {
			providerData, ok := data[p].(map[string]interface{})
			if !ok || providerData == nil {
				if p == "konachan_com" {
					t.Skipf("[%s] Skipping GraphQL test because provider data is null (likely CF block without FlareSolverr)", p)
				}
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
			var posts []interface{}
			var activeTag string

			if p1, ok := providerData["p1"].([]interface{}); ok && len(p1) > 0 {
				posts = p1
				activeTag = "yuri"
			} else if p2, ok := providerData["p2"].([]interface{}); ok && len(p2) > 0 {
				posts = p2
				activeTag = "1girl"
			} else if p3, ok := providerData["p3"].([]interface{}); ok && len(p3) > 0 {
				posts = p3
				activeTag = "bikini"
			}

			if len(posts) == 0 {
				if p == "konachan_com" {
					t.Logf("[%s] Warning: No posts found via GraphQL, likely due to Cloudflare 403. Skipping.", p)
					return
				}
				t.Fatalf("[%s] Warning: No posts found for tags yuri, 1girl, or bikini", p)
			}
			t.Logf("[%s] GraphQL Get Posts successful with tag '%s'. Fetched %d posts.", p, activeTag, len(posts))

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

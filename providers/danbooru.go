package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"matoi/config"
	"matoi/models"
)

// DanbooruPost represents a raw post response from the Danbooru API.
type DanbooruPost struct {
	ID             int    `json:"id"`
	FileURL        string `json:"file_url"`
	LargeFileURL   string `json:"large_file_url"`
	PreviewFileURL string `json:"preview_file_url"`
	TagString      string `json:"tag_string"`
	Rating         string `json:"rating"`
	Score          int    `json:"score"`
	MD5            string `json:"md5"`
	Source         string `json:"source"`
}

// DanbooruProvider handles fetching from Danbooru API and scraping for autocomplete.
type DanbooruProvider struct {
	Config *config.Config
}

// NewDanbooruProvider creates a new instance of DanbooruProvider.
func NewDanbooruProvider(cfg *config.Config) *DanbooruProvider {
	return &DanbooruProvider{
		Config: cfg,
	}
}

// FetchPosts fetches posts from the Danbooru API.
//
//nolint:gocyclo,gocognit // API parsing logic is inherently complex
func (p *DanbooruProvider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	reqURL, err := url.Parse(p.Config.DanbooruURL)
	if err != nil {
		return nil, fmt.Errorf("invalid danbooru url: %w", err)
	}

	q := reqURL.Query()
	if p.Config.DanbooruAPIID != "" && p.Config.DanbooruAPIKey != "" {
		q.Add("login", p.Config.DanbooruAPIID)
		q.Add("api_key", p.Config.DanbooruAPIKey)
	}

	if tags != "" {
		q.Add("tags", tags)
	}

	if limit <= 0 {
		limit = p.Config.DanbooruReturnLmt
	}
	q.Add("limit", strconv.Itoa(limit))
	q.Add("page", strconv.Itoa(page))

	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", p.Config.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from danbooru: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("danbooru api returned status code %d", resp.StatusCode)
	}

	var rawPosts []DanbooruPost
	if err := json.NewDecoder(resp.Body).Decode(&rawPosts); err != nil {
		return nil, fmt.Errorf("failed to decode danbooru response: %w", err)
	}

	posts := []models.Post{}
	for i := range rawPosts {
		rp := &rawPosts[i]
		// Skip posts that don't have file_url (Danbooru sometimes requires Gold account for certain files)
		if rp.FileURL == "" && rp.LargeFileURL == "" {
			continue
		}

		postTags := strings.Fields(rp.TagString)
		fileURL := rp.FileURL
		if fileURL == "" {
			fileURL = rp.LargeFileURL
		}

		post := models.Post{
			ID:         rp.ID,
			FileURL:    fileURL,
			PreviewURL: rp.PreviewFileURL,
			SampleURL:  rp.LargeFileURL,
			Rating:     rp.Rating,
			Score:      rp.Score,
			Source:     rp.Source,
			Tags:       postTags,
			Link:       fmt.Sprintf("https://danbooru.donmai.us/posts/%d", rp.ID),
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// QueryCompletion fetches tag autocomplete suggestions from Danbooru using tags.json.
func (p *DanbooruProvider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	searchURL := fmt.Sprintf("https://danbooru.donmai.us/tags.json?search[name_matches]=*%s*&search[order]=count&limit=15", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", p.Config.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("danbooru tags.json returned status %d", resp.StatusCode)
	}

	var rawTags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawTags); err != nil {
		return nil, fmt.Errorf("failed to parse danbooru tags.json: %w", err)
	}

	tags := []string{}
	for _, rt := range rawTags {
		if rt.Name != "" {
			tags = append(tags, rt.Name)
		}
	}

	return tags, nil
}

package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"matoi/config"
	"matoi/models"
)

// FurbooruImage represents a raw image response from the Furbooru API.
type FurbooruImage struct {
	ID              int               `json:"id"`
	Tags            []string          `json:"tags"`
	SourceURLs      []string          `json:"source_urls"`
	Representations map[string]string `json:"representations"`
	Score           int               `json:"score"`
}

// FurbooruResponse represents the top-level raw response from Furbooru search endpoint.
type FurbooruResponse struct {
	Images []FurbooruImage `json:"images"`
}

// FurbooruProvider handles fetching from Furbooru API.
type FurbooruProvider struct {
	Config *config.Config
}

// NewFurbooruProvider creates a new instance of FurbooruProvider.
func NewFurbooruProvider(cfg *config.Config) *FurbooruProvider {
	return &FurbooruProvider{
		Config: cfg,
	}
}

// FetchPosts fetches posts from the Furbooru API.
//
//nolint:gocognit,gocyclo // mapping logic inherently requires high cognitive complexity
func (p *FurbooruProvider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	baseURL := p.Config.FurbooruURL
	if baseURL == "" {
		baseURL = "https://furbooru.org/api/v1"
	}
	endpointURL := baseURL + "/json/search/images"

	reqURL, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("invalid furbooru url: %w", err)
	}

	q := reqURL.Query()
	if tags != "" {
		q.Add("q", tags)
	}

	if limit <= 0 {
		limit = p.Config.FurbooruReturnLmt
	}
	q.Add("per_page", strconv.Itoa(limit))
	q.Add("page", strconv.Itoa(page))

	if p.Config.FurbooruAPIKey != "" {
		q.Add("key", p.Config.FurbooruAPIKey)
	}

	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := p.Config.UserAgent
	if userAgent == "" {
		userAgent = "Matoi/1.0"
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from furbooru: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("furbooru api returned status code %d", resp.StatusCode)
	}

	var rawResp FurbooruResponse
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode furbooru response: %w", err)
	}

	return mapFurbooruImages(rawResp.Images), nil
}

func mapFurbooruImages(images []FurbooruImage) []models.Post {
	posts := make([]models.Post, 0, len(images))
	for i := range images {
		rp := &images[i]

		fullURL := rp.Representations["full"]
		// Skip if there's no full image URL
		if fullURL == "" {
			continue
		}

		sampleURL := rp.Representations["large"]
		if sampleURL == "" {
			sampleURL = fullURL
		}

		previewURL := rp.Representations["thumb"]
		if previewURL == "" {
			previewURL = rp.Representations["thumb_small"]
		}

		source := ""
		if len(rp.SourceURLs) > 0 {
			source = rp.SourceURLs[0]
		}

		rating := "safe"
		for _, tag := range rp.Tags {
			if tag == "explicit" || tag == "questionable" || tag == "suggestive" {
				rating = tag
				break
			}
		}

		post := models.Post{
			ID:         rp.ID,
			FileURL:    fullURL,
			PreviewURL: previewURL,
			SampleURL:  sampleURL,
			Rating:     rating,
			Score:      rp.Score,
			Source:     source,
			Tags:       rp.Tags,
			Link:       fmt.Sprintf("https://furbooru.org/images/%d", rp.ID),
		}
		posts = append(posts, post)
	}

	return posts
}

// QueryCompletion fetches tag autocomplete suggestions from Furbooru using /search/tags endpoint.
func (p *FurbooruProvider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	baseURL := p.Config.FurbooruURL
	if baseURL == "" {
		baseURL = "https://furbooru.org/api/v1"
	}
	searchURL := fmt.Sprintf("%s/json/search/tags?q=*%s*&per_page=15", baseURL, url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := p.Config.UserAgent
	if userAgent == "" {
		userAgent = "Matoi/1.0"
	}
	req.Header.Set("User-Agent", userAgent)

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
		return nil, fmt.Errorf("furbooru search tags returned status %d", resp.StatusCode)
	}

	var rawTagsResp struct {
		Tags []struct {
			Name string `json:"name"`
		} `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawTagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse furbooru tags response: %w", err)
	}

	tags := []string{}
	for _, rt := range rawTagsResp.Tags {
		if rt.Name != "" {
			tags = append(tags, rt.Name)
		}
	}

	return tags, nil
}

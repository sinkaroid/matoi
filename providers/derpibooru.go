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

// DerpibooruImage represents a raw image response from the Derpibooru API.
type DerpibooruImage struct {
	ID              int               `json:"id"`
	Tags            []string          `json:"tags"`
	SourceURLs      []string          `json:"source_urls"`
	Representations map[string]string `json:"representations"`
	Score           int               `json:"score"`
}

// DerpibooruResponse represents the top-level raw response from Derpibooru search endpoint.
type DerpibooruResponse struct {
	Images []DerpibooruImage `json:"images"`
}

// DerpibooruProvider handles fetching from Derpibooru API.
type DerpibooruProvider struct {
	Config *config.Config
}

// NewDerpibooruProvider creates a new instance of DerpibooruProvider.
func NewDerpibooruProvider(cfg *config.Config) *DerpibooruProvider {
	return &DerpibooruProvider{
		Config: cfg,
	}
}

// FetchPosts fetches posts from the Derpibooru API.
//
//nolint:gocognit,gocyclo // mapping logic inherently requires high cognitive complexity
func (p *DerpibooruProvider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	reqURL, err := url.Parse("https://derpibooru.org/api/v1/json/search/images")
	if err != nil {
		return nil, fmt.Errorf("invalid derpibooru url: %w", err)
	}

	q := reqURL.Query()
	if tags != "" {
		q.Add("q", tags)
	}

	if limit <= 0 {
		limit = p.Config.PostRestReturnLimit
	}
	q.Add("per_page", strconv.Itoa(limit))
	q.Add("page", strconv.Itoa(page))

	if p.Config.DerpibooruAPIKey != "" {
		q.Add("key", p.Config.DerpibooruAPIKey)
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
		return nil, fmt.Errorf("failed to fetch from derpibooru: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("derpibooru api returned status code %d", resp.StatusCode)
	}

	var rawResp DerpibooruResponse
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode derpibooru response: %w", err)
	}

	return mapDerpibooruImages(rawResp.Images), nil
}

//nolint:gocyclo,gocognit // mapping logic inherently requires high cognitive complexity
func mapDerpibooruImages(images []DerpibooruImage) []models.Post {
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
			if isDerpibooruRating(tag) {
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
			Link:       fmt.Sprintf("https://derpibooru.org/images/%d", rp.ID),
		}
		posts = append(posts, post)
	}

	return posts
}

func isDerpibooruRating(tag string) bool {
	switch tag {
	case "explicit", "questionable", "suggestive", "grimdark", "semi-grimdark", "grotesque":
		return true
	default:
		return false
	}
}

// QueryCompletion fetches tag autocomplete suggestions from Derpibooru using /search/tags endpoint.
func (p *DerpibooruProvider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	searchURL := fmt.Sprintf("https://derpibooru.org/api/v1/json/search/tags?q=*%s*&per_page=15", url.QueryEscape(query))

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
		return nil, fmt.Errorf("derpibooru search tags returned status %d", resp.StatusCode)
	}

	var rawTagsResp struct {
		Tags []struct {
			Name string `json:"name"`
		} `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawTagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse derpibooru tags response: %w", err)
	}

	tags := []string{}
	for _, rt := range rawTagsResp.Tags {
		if rt.Name != "" {
			tags = append(tags, rt.Name)
		}
	}

	return tags, nil
}

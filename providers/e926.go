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

// E926Post represents a raw post response from the E926 API.
type E926Post struct {
	ID   int `json:"id"`
	File struct {
		URL string `json:"url"`
	} `json:"file"`
	Preview struct {
		URL string `json:"url"`
	} `json:"preview"`
	Sample struct {
		URL string `json:"url"`
	} `json:"sample"`
	Rating string `json:"rating"`
	Score  struct {
		Total int `json:"total"`
	} `json:"score"`
	Sources []string `json:"sources"`
	Tags    struct {
		General   []string `json:"general"`
		Artist    []string `json:"artist"`
		Character []string `json:"character"`
		Copyright []string `json:"copyright"`
		Species   []string `json:"species"`
		Meta      []string `json:"meta"`
	} `json:"tags"`
}

// E926Response represents the top-level raw response from E926.
type E926Response struct {
	Posts []E926Post `json:"posts"`
}

// E926Provider handles fetching from E926 API and tags query completion.
type E926Provider struct {
	Config *config.Config
}

// NewE926Provider creates a new instance of E926Provider.
func NewE926Provider(cfg *config.Config) *E926Provider {
	return &E926Provider{
		Config: cfg,
	}
}

// FetchPosts fetches posts from the E926 API.
//
//nolint:gocognit,gocyclo // mapping logic inherently requires high cognitive complexity
func (p *E926Provider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	reqURL, err := url.Parse("https://e926.net/posts.json")
	if err != nil {
		return nil, fmt.Errorf("invalid e926 url: %w", err)
	}

	q := reqURL.Query()
	if tags != "" {
		q.Add("tags", tags)
	}

	if limit <= 0 {
		limit = p.Config.PostRestReturnLimit
	}
	q.Add("limit", strconv.Itoa(limit))
	q.Add("page", strconv.Itoa(page))

	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if p.Config.E926APIID != "" && p.Config.E926APIKey != "" {
		req.SetBasicAuth(p.Config.E926APIID, p.Config.E926APIKey)
	}

	// User-Agent is strictly required
	userAgent := p.Config.UserAgent
	if userAgent == "" {
		userAgent = "Matoi/1.0"
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from e926: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("e926 api returned status code %d", resp.StatusCode)
	}

	var rawResp E926Response
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode e926 response: %w", err)
	}

	posts := []models.Post{}
	for i := range rawResp.Posts {
		rp := &rawResp.Posts[i]

		// Skip posts that don't have file_url
		if rp.File.URL == "" && rp.Sample.URL == "" {
			continue
		}

		// Combine tags
		var postTags []string
		postTags = append(postTags, rp.Tags.General...)
		postTags = append(postTags, rp.Tags.Artist...)
		postTags = append(postTags, rp.Tags.Character...)
		postTags = append(postTags, rp.Tags.Copyright...)
		postTags = append(postTags, rp.Tags.Species...)
		postTags = append(postTags, rp.Tags.Meta...)

		fileURL := rp.File.URL
		if fileURL == "" {
			fileURL = rp.Sample.URL
		}

		source := ""
		if len(rp.Sources) > 0 {
			source = rp.Sources[0]
		}

		post := models.Post{
			ID:         rp.ID,
			FileURL:    fileURL,
			PreviewURL: rp.Preview.URL,
			SampleURL:  rp.Sample.URL,
			Rating:     rp.Rating,
			Score:      rp.Score.Total,
			Source:     source,
			Tags:       postTags,
			Link:       fmt.Sprintf("https://e926.net/posts/%d", rp.ID),
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// QueryCompletion fetches tag autocomplete suggestions from E926 using tags.json.
func (p *E926Provider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	searchURL := fmt.Sprintf("https://e926.net/tags.json?search[name_matches]=*%s*&search[order]=count&limit=15", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// User-Agent is strictly required
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
		return nil, fmt.Errorf("e926 tags.json returned status %d", resp.StatusCode)
	}

	var rawTags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawTags); err != nil {
		return nil, fmt.Errorf("failed to parse e926 tags.json: %w", err)
	}

	tags := []string{}
	for _, rt := range rawTags {
		if rt.Name != "" {
			tags = append(tags, rt.Name)
		}
	}

	return tags, nil
}

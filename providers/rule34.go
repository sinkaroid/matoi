// Package providers contains all the external booru API binding implementations.
package providers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"matoi/config"
	"matoi/models"

	"github.com/PuerkitoBio/goquery"
)

// Rule34Post represents a raw post response from the Rule34 API.
type Rule34Post struct {
	ID         int    `json:"id"`
	FileURL    string `json:"file_url"`
	PreviewURL string `json:"preview_url"`
	SampleURL  string `json:"sample_url"`
	Tags       string `json:"tags"`
	Rating     string `json:"rating"`
	Score      int    `json:"score"`
	Directory  int    `json:"directory"`
	Hash       string `json:"hash"`
	Source     string `json:"source"`
	Image      string `json:"image"`
	Sample     bool   `json:"sample"`
}

// Rule34Provider manages fetching posts from Rule34.
type Rule34Provider struct {
	Cfg *config.Config
}

// NewRule34Provider creates a new Rule34 provider instance.
func NewRule34Provider(cfg *config.Config) *Rule34Provider {
	return &Rule34Provider{Cfg: cfg}
}

// FetchPosts fetches posts from the Rule34 API based on tags, limit, and page.
func (p *Rule34Provider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
	// Disable TLS verification to handle simulated time (2026) certificate validation failures
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	reqURL, err := p.buildURL(tags, limit, page)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from upstream: %d", resp.StatusCode)
	}

	var rawPosts []Rule34Post
	if err := json.NewDecoder(resp.Body).Decode(&rawPosts); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return mapPosts(rawPosts), nil
}

// QueryCompletion fetches Eiyuu-style tag autocomplete suggestions from Rule34.
func (p *Rule34Provider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	// Eiyuu uses the web site, not the API, for autocomplete scraping.
	urlStr := fmt.Sprintf("https://rule34.xxx/index.php?page=tags&s=list&tags=*%s*&sort=desc&order_by=index_count", query)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if p.Cfg.UserAgent != "" {
		req.Header.Set("User-Agent", p.Cfg.UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from upstream: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return parseAutocompleteTags(doc), nil
}

func parseAutocompleteTags(doc *goquery.Document) []string {
	tags := []string{}
	doc.Find("table.highlightable a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || !strings.Contains(href, "&tags=") {
			return
		}

		parts := strings.Split(href, "&tags=")
		if len(parts) > 1 {
			decodedURL, err := url.QueryUnescape(parts[1])
			if err != nil {
				decodedURL = parts[1] // fallback
			}
			tags = append(tags, html.UnescapeString(decodedURL))
		}
	})
	return tags
}

func (p *Rule34Provider) buildURL(tags string, limit, page int) (string, error) {
	u, err := url.Parse(p.Cfg.Rule34URL)
	if err != nil {
		return "", fmt.Errorf("failed to parse Rule34 URL: %w", err)
	}

	q := u.Query()
	q.Set("page", "dapi")
	q.Set("s", "post")
	q.Set("q", "index")
	q.Set("json", "1")
	q.Set("tags", tags)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("pid", strconv.Itoa(page-1)) // 0-based page index

	if p.Cfg.Rule34APIKey != "" {
		q.Set("api_key", p.Cfg.Rule34APIKey)
	}
	if p.Cfg.Rule34APIID != "" {
		q.Set("user_id", p.Cfg.Rule34APIID)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func mapRating(rawRating string) string {
	switch strings.ToLower(rawRating) {
	case "explicit", "e":
		return "e"
	case "questionable", "q":
		return "q"
	case "safe", "s":
		return "s"
	default:
		return "s"
	}
}

func mapPosts(rawPosts []Rule34Post) []models.Post {
	posts := make([]models.Post, len(rawPosts))
	for i := range rawPosts {
		rp := &rawPosts[i]
		posts[i] = models.Post{
			ID:         rp.ID,
			Directory:  strconv.Itoa(rp.Directory),
			FileURL:    rp.FileURL,
			PreviewURL: rp.PreviewURL,
			SampleURL:  rp.SampleURL,
			Rating:     mapRating(rp.Rating),
			Score:      rp.Score,
			Source:     rp.Source,
			Image:      rp.Image,
			Tags:       strings.Fields(rp.Tags),
			Link:       fmt.Sprintf("https://rule34.xxx/index.php?page=post&s=view&id=%d", rp.ID),
		}
	}
	return posts
}

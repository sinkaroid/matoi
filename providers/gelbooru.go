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

// GelbooruPost represents a raw post response from the Gelbooru API.
type GelbooruPost struct {
	ID         int    `json:"id"`
	FileURL    string `json:"file_url"`
	PreviewURL string `json:"preview_url"`
	SampleURL  string `json:"sample_url"`
	Tags       string `json:"tags"`
	Rating     string `json:"rating"`
	Score      int    `json:"score"`
	Directory  string `json:"directory"`
	Source     string `json:"source"`
	Image      string `json:"image"`
}

// GelbooruResponse represents the top-level raw response from Gelbooru.
type GelbooruResponse struct {
	Attributes map[string]interface{} `json:"@attributes"`
	Post       []GelbooruPost         `json:"post"`
}

// GelbooruProvider manages fetching posts from Gelbooru.
type GelbooruProvider struct {
	Cfg *config.Config
}

// NewGelbooruProvider creates a new Gelbooru provider instance.
func NewGelbooruProvider(cfg *config.Config) *GelbooruProvider {
	return &GelbooruProvider{Cfg: cfg}
}

// FetchPosts fetches posts from the Gelbooru API based on tags, limit, and page.
func (p *GelbooruProvider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
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

	var rawResponse GelbooruResponse
	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.mapPosts(rawResponse.Post), nil
}

// QueryCompletion fetches Eiyuu-style tag autocomplete suggestions from Gelbooru.
func (p *GelbooruProvider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	urlStr := fmt.Sprintf("https://gelbooru.com/index.php?page=tags&s=list&tags=*%s*&sort=desc&order_by=index_count", query)

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

	return p.parseAutocompleteTags(doc), nil
}

func (p *GelbooruProvider) parseAutocompleteTags(doc *goquery.Document) []string {
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

func (p *GelbooruProvider) buildURL(tags string, limit, page int) (string, error) {
	u, err := url.Parse("https://gelbooru.com/index.php")
	if err != nil {
		return "", fmt.Errorf("failed to parse Gelbooru URL: %w", err)
	}

	q := u.Query()
	q.Set("page", "dapi")
	q.Set("s", "post")
	q.Set("q", "index")
	q.Set("json", "1")
	q.Set("tags", tags)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("pid", strconv.Itoa(page-1)) // 0-based page index

	if p.Cfg.GelbooruAPIKey != "" {
		q.Set("api_key", p.Cfg.GelbooruAPIKey)
	}
	if p.Cfg.GelbooruUserID != "" {
		q.Set("user_id", p.Cfg.GelbooruUserID)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p *GelbooruProvider) mapRating(rawRating string) string {
	switch strings.ToLower(rawRating) {
	case "explicit", "e":
		return "e"
	case "questionable", "q":
		return "q"
	case "safe", "s", "general":
		return "s"
	default:
		return "s"
	}
}

func (p *GelbooruProvider) mapPosts(rawPosts []GelbooruPost) []models.Post {
	posts := make([]models.Post, len(rawPosts))
	for i := range rawPosts {
		rp := &rawPosts[i]

		posts[i] = models.Post{
			ID:         rp.ID,
			Directory:  rp.Directory,
			FileURL:    rp.FileURL,
			PreviewURL: rp.PreviewURL,
			SampleURL:  rp.SampleURL,
			Rating:     p.mapRating(rp.Rating),
			Score:      rp.Score,
			Source:     rp.Source,
			Image:      rp.Image,
			Tags:       strings.Fields(rp.Tags),
			Link:       fmt.Sprintf("https://gelbooru.com/index.php?page=post&s=view&id=%d", rp.ID),
		}
	}
	return posts
}

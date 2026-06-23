package providers

import (
	"bytes"
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

// KonachanComPost represents a raw post response from the KonachanCom JSON API.
type KonachanComPost struct {
	ID         int    `json:"id"`
	Tags       string `json:"tags"`
	FileURL    string `json:"file_url"`
	PreviewURL string `json:"preview_url"`
	SampleURL  string `json:"sample_url"`
	Rating     string `json:"rating"`
	Score      int    `json:"score"`
	Source     string `json:"source"`
}

// KonachanComProvider manages fetching posts from KonachanCom.
type KonachanComProvider struct {
	Cfg *config.Config
}

// NewKonachanComProvider creates a new KonachanCom provider instance.
func NewKonachanComProvider(cfg *config.Config) *KonachanComProvider {
	return &KonachanComProvider{Cfg: cfg}
}

// FetchPosts fetches posts from the KonachanCom API based on tags, limit, and page.
func (p *KonachanComProvider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
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

	if p.Cfg.FlareSolverrURL != "" {
		return p.fetchPostsViaFlareSolverr(ctx, reqURL)
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

	var rawPosts []KonachanComPost
	if err := json.NewDecoder(resp.Body).Decode(&rawPosts); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return mapKonachanComPosts(rawPosts), nil
}

func (p *KonachanComProvider) fetchPostsViaFlareSolverr(ctx context.Context, targetURL string) ([]models.Post, error) {
	respBody, err := p.doFlareSolverrRequest(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	// FlareSolverr returns the full HTML page. JSON is usually wrapped in <pre> tags by Chrome.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(respBody))
	if err != nil {
		return nil, fmt.Errorf("failed to parse flaresolverr html: %w", err)
	}

	jsonText := doc.Find("body").Text()
	if jsonText == "" {
		jsonText = respBody // fallback
	}

	var rawPosts []KonachanComPost
	if err := json.Unmarshal([]byte(jsonText), &rawPosts); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from flaresolverr: %w", err)
	}

	return mapKonachanComPosts(rawPosts), nil
}

// QueryCompletion fetches Eiyuu-style tag autocomplete suggestions from KonachanCom.
func (p *KonachanComProvider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	urlStr := fmt.Sprintf("https://konachan.com/tag?name=*%s*", url.QueryEscape(query))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	if p.Cfg.FlareSolverrURL != "" {
		return p.queryCompletionViaFlareSolverr(ctx, urlStr)
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

	return parseKonachanComAutocompleteTags(doc), nil
}

func (p *KonachanComProvider) queryCompletionViaFlareSolverr(ctx context.Context, targetURL string) ([]string, error) {
	respBody, err := p.doFlareSolverrRequest(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(respBody))
	if err != nil {
		return nil, fmt.Errorf("failed to parse flaresolverr html: %w", err)
	}

	return parseKonachanComAutocompleteTags(doc), nil
}

type flareSolverrResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Response string `json:"response"`
		Status   int    `json:"status"`
	} `json:"solution"`
}

func (p *KonachanComProvider) doFlareSolverrRequest(ctx context.Context, targetURL string) (string, error) {
	payload := map[string]interface{}{
		"cmd":        "request.get",
		"url":        targetURL,
		"maxTimeout": 60000,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal flaresolverr payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Cfg.FlareSolverrURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create flaresolverr request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 65 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("flaresolverr request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("flaresolverr returned status %d", resp.StatusCode)
	}

	var fsResp flareSolverrResponse
	if err := json.NewDecoder(resp.Body).Decode(&fsResp); err != nil {
		return "", fmt.Errorf("failed to decode flaresolverr JSON: %w", err)
	}

	if fsResp.Status != "ok" {
		return "", fmt.Errorf("flaresolverr failed to solve: %s", fsResp.Message)
	}

	if fsResp.Solution.Status != http.StatusOK {
		return "", fmt.Errorf("upstream returned status %d via flaresolverr", fsResp.Solution.Status)
	}

	return fsResp.Solution.Response, nil
}

func parseKonachanComAutocompleteTags(doc *goquery.Document) []string {
	tags := []string{}
	doc.Find("table.highlightable td:nth-child(2) a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || !strings.Contains(href, "tags=") {
			return
		}

		parts := strings.Split(href, "tags=")
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

func (p *KonachanComProvider) buildURL(tags string, limit, page int) (string, error) {
	u, err := url.Parse("https://konachan.com/post.json")
	if err != nil {
		return "", fmt.Errorf("failed to parse KonachanCom URL: %w", err)
	}

	q := u.Query()
	q.Set("tags", tags)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("page", strconv.Itoa(page)) // KonachanCom uses 1-based page index

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func mapKonachanComRating(rawRating string) string {
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

func mapKonachanComPosts(rawPosts []KonachanComPost) []models.Post {
	posts := make([]models.Post, len(rawPosts))
	for i := range rawPosts {
		rp := &rawPosts[i]

		directory := ""
		image := ""
		if parsed, err := url.Parse(rp.FileURL); err == nil {
			parts := strings.Split(parsed.Path, "/")
			if len(parts) >= 3 {
				image = parts[len(parts)-1]
				directory = parts[len(parts)-2]
			}
		}

		posts[i] = models.Post{
			ID:         rp.ID,
			Directory:  directory,
			Image:      image,
			FileURL:    rp.FileURL,
			PreviewURL: rp.PreviewURL,
			SampleURL:  rp.SampleURL,
			Rating:     mapKonachanComRating(rp.Rating),
			Score:      rp.Score,
			Source:     rp.Source,
			Tags:       strings.Fields(rp.Tags),
			Link:       fmt.Sprintf("https://konachan.com/post/show/%d", rp.ID),
		}
	}
	return posts
}

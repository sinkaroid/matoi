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

// RealbooruProvider manages fetching posts from Realbooru.
// Since their API is broken, it uses HTML scraping.
type RealbooruProvider struct {
	Cfg *config.Config
}

// NewRealbooruProvider creates a new Realbooru provider instance.
func NewRealbooruProvider(cfg *config.Config) *RealbooruProvider {
	return &RealbooruProvider{Cfg: cfg}
}

// FetchPosts fetches posts from Realbooru by scraping the HTML DOM.
func (p *RealbooruProvider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	reqURL, err := p.buildURL(tags, page)
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

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var posts []models.Post
	doc.Find("div.col.thumb").Each(func(_ int, s *goquery.Selection) {
		if p := parseRealbooruPost(s); p != nil {
			posts = append(posts, *p)
		}
	})

	// Truncate to requested limit if necessary
	if len(posts) > limit {
		posts = posts[:limit]
	}

	return posts, nil
}

// QueryCompletion fetches tag autocomplete suggestions using Realbooru's native JSON autocomplete endpoint.
func (p *RealbooruProvider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	urlStr := fmt.Sprintf("https://realbooru.com/index.php?page=autocomplete&term=%s", url.QueryEscape(query))

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

	var rawTags []string
	if err := json.NewDecoder(resp.Body).Decode(&rawTags); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	tags := make([]string, 0, len(rawTags))
	for _, rawTag := range rawTags {
		decoded, err := url.QueryUnescape(rawTag)
		if err != nil {
			decoded = rawTag
		}
		tags = append(tags, html.UnescapeString(decoded))
	}

	return tags, nil
}

func (p *RealbooruProvider) buildURL(tags string, page int) (string, error) {
	u, err := url.Parse("https://realbooru.com/index.php")
	if err != nil {
		return "", fmt.Errorf("failed to parse Realbooru URL: %w", err)
	}

	q := u.Query()
	q.Set("page", "post")
	q.Set("s", "list")
	q.Set("tags", tags)

	// Realbooru uses pid for offset based on a fixed limit of 42 per page
	pid := (page - 1) * 42
	if pid < 0 {
		pid = 0
	}
	q.Set("pid", strconv.Itoa(pid))

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseRealbooruPost(s *goquery.Selection) *models.Post {
	idStr, _ := s.Attr("id")
	idStr = strings.TrimPrefix(idStr, "s")

	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		return nil
	}

	img := s.Find("img")
	previewURL, _ := img.Attr("src")
	tagsStr, _ := img.Attr("title")

	// Split tags by comma, trim, and replace spaces with underscores to match canonical tag format
	tagsArray := []string{}
	if tagsStr != "" {
		for _, tag := range strings.Split(tagsStr, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tag = strings.ReplaceAll(tag, " ", "_")
				tagsArray = append(tagsArray, tag)
			}
		}
	}

	// Infer FileURL based on previewURL
	fileURL := strings.Replace(previewURL, "thumbnails/", "images/", 1)
	fileURL = strings.Replace(fileURL, "thumbnail_", "", 1)

	directory := ""
	imageName := ""
	if parsed, pErr := url.Parse(fileURL); pErr == nil {
		parts := strings.Split(parsed.Path, "/")
		if len(parts) >= 4 {
			imageName = parts[len(parts)-1]
			directory = parts[len(parts)-3] + "/" + parts[len(parts)-2]
		} else if len(parts) >= 3 {
			imageName = parts[len(parts)-1]
			directory = parts[len(parts)-2]
		}
	}

	return &models.Post{
		ID:         idInt,
		Directory:  directory,
		Image:      imageName,
		FileURL:    fileURL,
		PreviewURL: previewURL,
		SampleURL:  fileURL, // Assume same as fileURL since no sample available
		Rating:     "q",     // Default fallback since it's an adult site and tags don't expose it
		Score:      0,       // Cannot be scraped from thumbnail page easily
		Source:     "",
		Tags:       tagsArray,
		Link:       fmt.Sprintf("https://realbooru.com/index.php?page=post&s=view&id=%d", idInt),
	}
}

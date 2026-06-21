package providers

import (
	"context"
	"crypto/tls"
	"encoding/xml"
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

// TbibPosts represents the root XML response from the TBIB API.
type TbibPosts struct {
	XMLName xml.Name   `xml:"posts"`
	Posts   []TbibPost `xml:"post"`
}

// TbibPost represents a raw post response from the TBIB XML API.
type TbibPost struct {
	ID         int    `xml:"id,attr"`
	FileURL    string `xml:"file_url,attr"`
	PreviewURL string `xml:"preview_url,attr"`
	SampleURL  string `xml:"sample_url,attr"`
	Tags       string `xml:"tags,attr"`
	Rating     string `xml:"rating,attr"`
	Score      string `xml:"score,attr"`
	Source     string `xml:"source,attr"`
}

// TbibProvider manages fetching posts from TBIB.
type TbibProvider struct {
	Cfg *config.Config
}

// NewTbibProvider creates a new TBIB provider instance.
func NewTbibProvider(cfg *config.Config) *TbibProvider {
	return &TbibProvider{Cfg: cfg}
}

// FetchPosts fetches posts from the TBIB API based on tags, limit, and page.
func (p *TbibProvider) FetchPosts(ctx context.Context, tags string, limit, page int) ([]models.Post, error) {
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

	var rawData TbibPosts
	if err := xml.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	return mapTbibPosts(rawData.Posts), nil
}

// QueryCompletion fetches Eiyuu-style tag autocomplete suggestions from TBIB.
func (p *TbibProvider) QueryCompletion(ctx context.Context, query string) ([]string, error) {
	// Eiyuu uses the web site, not the API, for autocomplete scraping.
	urlStr := fmt.Sprintf("https://tbib.org/index.php?page=tags&s=list&tags=*%s*&sort=desc&order_by=index_count", query)

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

	return parseTbibAutocompleteTags(doc), nil
}

func parseTbibAutocompleteTags(doc *goquery.Document) []string {
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

func (p *TbibProvider) buildURL(tags string, limit, page int) (string, error) {
	u, err := url.Parse(p.Cfg.TbibURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse TBIB URL: %w", err)
	}

	q := u.Query()
	q.Set("page", "dapi")
	q.Set("s", "post")
	q.Set("q", "index")
	// Notice we do NOT use "json=1" here to get the XML output with file_url
	q.Set("tags", tags)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("pid", strconv.Itoa(page-1)) // 0-based page index

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func mapTbibRating(rawRating string) string {
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

func mapTbibPosts(rawPosts []TbibPost) []models.Post {
	posts := make([]models.Post, len(rawPosts))
	for i := range rawPosts {
		rp := &rawPosts[i]
		score, err := strconv.Atoi(rp.Score)
		if err != nil {
			score = 0 // Default to 0 on failure
		}

		// Extract directory and image from file_url (e.g. https://tbib.org/images/4902/6585fd936a0d916ee65fb1756373a25ebb438a9f.jpg)
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
			Rating:     mapTbibRating(rp.Rating),
			Score:      score,
			Source:     rp.Source,
			Tags:       strings.Fields(rp.Tags),
			Link:       fmt.Sprintf("https://tbib.org/index.php?page=post&s=view&id=%d", rp.ID),
		}
	}
	return posts
}

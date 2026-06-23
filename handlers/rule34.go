// Package handlers contains all the HTTP endpoints handlers for the application.
package handlers

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"matoi/cache"
	"matoi/models"
	"matoi/providers"

	"github.com/gofiber/fiber/v3"
)

// Rule34Handler exposes the Rule34 provider endpoints.
type Rule34Handler struct {
	provider *providers.Rule34Provider
}

// NewRule34Handler creates a new Rule34 handler instance.
func NewRule34Handler(p *providers.Rule34Provider) *Rule34Handler {
	return &Rule34Handler{provider: p}
}

// Rule34Response defines the JSON response schema for posts query.
type Rule34Response struct {
	Success  bool          `json:"success"`
	Provider string        `json:"provider"`
	Count    int           `json:"count"`
	Posts    []models.Post `json:"posts"`
}

// GetPosts returns posts from Rule34, checking Redis cache first.
//
//	@Summary	Get posts from Rule34
//	@Tags		rule34
//	@Produce	json
//	@Param		tags	query		string	false	"Space-separated tags"
//	@Param		limit	query		int		false	"Max results (default 20, max 100)"
//	@Param		page	query		int		false	"Page number (default 1)"
//	@Param		shuffle	query		bool	false	"Shuffle the results"
//	@Success	200		{object}	Rule34Response
//	@Failure	502		{object}	map[string]string	"Upstream fetch failed"
//	@Security	ApiKeyAuth
//	@Router		/api/rule34/posts [get]
//
//nolint:gocyclo,gocognit // Parsing request parameters makes this function complex
func (h *Rule34Handler) GetPosts(c fiber.Ctx) error {
	tags := c.Query("tags", "")

	limitStr := c.Query("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	maxLimit := h.provider.Cfg.PostRestReturnLimit
	if maxLimit <= 0 {
		maxLimit = 100
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	pageStr := c.Query("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	// Cache key format: rule34:posts:tags_limit_page
	cacheKey := fmt.Sprintf("rule34:posts:%s:%d:%d", tags, limit, page)
	ctx := c.Context()

	var posts []models.Post
	found, err := cache.Get(ctx, cacheKey, &posts)
	if err == nil && found {
		c.Locals("source", "CACHE")
		h.resolveMatoiURLs(c, posts)

		if c.Query("shuffle", "false") == "true" && len(posts) > 0 {
			rand.Shuffle(len(posts), func(i, j int) {
				posts[i], posts[j] = posts[j], posts[i]
			})
		}

		if len(posts) == 0 {
			return c.Status(http.StatusNotFound).JSON(Rule34Response{
				Success:  false,
				Provider: "rule34",
				Count:    0,
				Posts:    []models.Post{},
			})
		}
		return c.Status(http.StatusOK).JSON(Rule34Response{
			Success:  true,
			Provider: "rule34",
			Count:    len(posts),
			Posts:    posts,
		})
	}

	// Fetch from upstream
	posts, err = h.provider.FetchPosts(c.Context(), tags, limit, page)
	if err != nil {
		// Do not cache if not 200 OK (upstream request failed or returned non-200)
		return fiber.NewError(http.StatusBadGateway, fmt.Sprintf("Upstream fetch failed: %v", err))
	}

	// Only cache if we got results (do not cache empty 404s)
	if len(posts) > 0 {
		ttl := h.provider.Cfg.RedisExpireCache
		if setErr := cache.Set(ctx, cacheKey, posts, ttl); setErr != nil {
			// Log the error but do not fail the request
			_ = setErr
		}
	}

	c.Locals("source", "FETCH")
	h.resolveMatoiURLs(c, posts)

	if c.Query("shuffle", "false") == "true" && len(posts) > 0 {
		rand.Shuffle(len(posts), func(i, j int) {
			posts[i], posts[j] = posts[j], posts[i]
		})
	}

	if len(posts) == 0 {
		return c.Status(http.StatusNotFound).JSON(Rule34Response{
			Success:  false,
			Provider: "rule34",
			Count:    0,
			Posts:    []models.Post{},
		})
	}

	return c.Status(http.StatusOK).JSON(Rule34Response{
		Success:  true,
		Provider: "rule34",
		Count:    len(posts),
		Posts:    posts,
	})
}

// ProxyMedia fetches and streams blocked Rule34 media.
//
//	@Summary	Proxy and stream Rule34 media
//	@Tags		rule34
//	@Param		url	query	string	true	"Encoded Rule34 media URL"
//	@Success	200	"Streams the media file"
//	@Failure	400	{object}	map[string]string	"Invalid parameters"
//	@Failure	502	{object}	map[string]string	"Failed to fetch media"
//	@Router		/api/rule34/media [get]
func (h *Rule34Handler) ProxyMedia(c fiber.Ctx) error {
	mediaURL := c.Query("url", "")
	if mediaURL == "" {
		return fiber.NewError(http.StatusBadRequest, "Missing url parameter")
	}

	if !isValidMediaDomain(mediaURL) {
		return fiber.NewError(http.StatusBadRequest, "Invalid media source domain")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
			ResponseHeaderTimeout: 15 * time.Second,
		},
		// No Timeout here — total timeout would kill large file streaming mid-transfer
	}

	req, err := http.NewRequestWithContext(c.Context(), http.MethodGet, mediaURL, http.NoBody)
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, "Failed to create proxy request")
	}
	req.Header.Set("User-Agent", h.provider.Cfg.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fiber.NewError(http.StatusBadGateway, "Failed to fetch media from upstream")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fiber.NewError(http.StatusBadGateway, fmt.Sprintf("Upstream media returned status %d", resp.StatusCode))
	}

	// Set content headers from upstream
	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		c.Set("Content-Type", contentType)
	}
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		c.Set("Content-Length", contentLength)
	}

	// Synchronously copy the response body to the client to avoid premature close of resp.Body
	if _, err := io.Copy(c.Response().BodyWriter(), resp.Body); err != nil {
		return fiber.NewError(http.StatusBadGateway, "Failed to stream media content")
	}

	return nil
}

func isValidMediaDomain(mediaURL string) bool {
	parsedURL, err := url.Parse(mediaURL)
	if err != nil {
		return false
	}
	return strings.HasSuffix(parsedURL.Host, ".rule34.xxx") || parsedURL.Host == "rule34.xxx"
}

func (h *Rule34Handler) resolveMatoiURLs(c fiber.Ctx, posts []models.Post) {
	baseURL := h.provider.Cfg.ResolverURL
	if baseURL == "" {
		baseURL = c.BaseURL()
	}

	// Ensure no trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	for i := range posts {
		p := &posts[i]
		if p.FileURL != "" {
			p.MatoiFileURL = fmt.Sprintf("%s/api/rule34/media?url=%s", baseURL, url.QueryEscape(p.FileURL))
		}
		if p.PreviewURL != "" {
			p.MatoiPreviewURL = fmt.Sprintf("%s/api/rule34/media?url=%s", baseURL, url.QueryEscape(p.PreviewURL))
		}
		if p.SampleURL != "" {
			p.MatoiSampleURL = fmt.Sprintf("%s/api/rule34/media?url=%s", baseURL, url.QueryEscape(p.SampleURL))
		}
	}
}

// QueryCompletionResponse defines the JSON response schema for tag autocomplete.
type QueryCompletionResponse struct {
	Success bool     `json:"success"`
	Tags    []string `json:"tags"`
}

// QueryCompletion handles the /api/rule34/query_completion endpoint.
//
//	@Summary		Get tag completion from Rule34 (Eiyuu logic)
//	@Description	Scrapes autocomplete tags from Rule34 using wildcard matching.
//	@Tags			rule34
//	@Produce		json
//	@Param			tags	query		string	true	"Tag query to autocomplete (e.g. jeanne)"
//	@Success		200		{object}	QueryCompletionResponse
//	@Failure		400		{object}	main.ErrorResponse
//	@Failure		502		{object}	main.ErrorResponse
//	@Security		ApiKeyAuth
//	@Router			/api/rule34/query_completion [get]
func (h *Rule34Handler) QueryCompletion(c fiber.Ctx) error {
	query := c.Query("tags", "")
	if query == "" {
		return fiber.NewError(http.StatusBadRequest, "tags query parameter is required")
	}

	// Fetch from upstream
	tags, err := h.provider.QueryCompletion(c.Context(), query)
	if err != nil {
		return fiber.NewError(http.StatusBadGateway, fmt.Sprintf("Upstream fetch failed: %v", err))
	}

	if len(tags) == 0 {
		return c.Status(http.StatusNotFound).JSON(QueryCompletionResponse{
			Success: false,
			Tags:    []string{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(QueryCompletionResponse{
		Success: true,
		Tags:    tags,
	})
}

package handlers

import (
	"fmt"
	"io"
	"math/rand/v2"
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

// XbooruHandler handles HTTP requests for Xbooru API endpoints.
type XbooruHandler struct {
	Provider *providers.XbooruProvider
}

// NewXbooruHandler creates a new handler instance for Xbooru.
func NewXbooruHandler(p *providers.XbooruProvider) *XbooruHandler {
	return &XbooruHandler{Provider: p}
}

// GetPosts fetches posts from Xbooru with pagination and caching.
//
//	@Summary	Fetch Xbooru Posts
//	@Tags		xbooru
//	@Produce	json
//	@Param		tags	query		string	false	"Tags to search for"
//	@Param		limit	query		int		false	"Number of results (default 20, max 100)"
//	@Param		page	query		int		false	"Page number (default 1)"
//	@Param		shuffle	query		bool	false	"Randomize the order of the results"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]interface{}
//	@Security	ApiKeyAuth
//	@Router		/api/xbooru/posts [get]
//
//nolint:gocyclo,gocognit // Parsing request parameters makes this function complex
func (h *XbooruHandler) GetPosts(c fiber.Ctx) error {
	tags := c.Query("tags", "")
	limitStr := c.Query("limit", "20")
	pageStr := c.Query("page", "1")
	shuffleStr := c.Query("shuffle", "false")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	maxLimit, err := strconv.Atoi(h.Provider.Cfg.XbooruReturnLimit)
	if err != nil || maxLimit <= 0 {
		maxLimit = 100
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	shuffle := false
	if shuffleStr == "true" {
		shuffle = true
	}

	ctx := c.Context()
	cacheKey := fmt.Sprintf("xbooru:posts:%s:%d:%d", tags, limit, page)

	var posts []models.Post
	found, err := cache.Get(ctx, cacheKey, &posts)
	if err == nil && found {
		h.resolveMatoiURLs(c.BaseURL(), posts)
		if shuffle && len(posts) > 0 {
			rand.Shuffle(len(posts), func(i, j int) {
				posts[i], posts[j] = posts[j], posts[i]
			})
		}
		if len(posts) == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success":  false,
				"provider": "xbooru",
				"count":    0,
				"posts":    []models.Post{},
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":  true,
			"provider": "xbooru",
			"count":    len(posts),
			"posts":    posts,
		})
	}

	posts, err = h.Provider.FetchPosts(ctx, tags, limit, page)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("failed to fetch from Xbooru: %v", err))
	}

	if len(posts) > 0 {
		if setErr := cache.Set(ctx, cacheKey, posts, h.Provider.Cfg.RedisExpireCache); setErr != nil {
			_ = setErr
		}
	}

	h.resolveMatoiURLs(c.BaseURL(), posts)
	if shuffle && len(posts) > 0 {
		rand.Shuffle(len(posts), func(i, j int) {
			posts[i], posts[j] = posts[j], posts[i]
		})
	}

	if len(posts) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success":  false,
			"provider": "xbooru",
			"count":    0,
			"posts":    []models.Post{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":  true,
		"provider": "xbooru",
		"count":    len(posts),
		"posts":    posts,
	})
}

// QueryCompletion fetches Eiyuu-style tag autocomplete suggestions. NO CACHE.
//
//	@Summary	Xbooru Tag Autocomplete
//	@Tags		xbooru
//	@Produce	json
//	@Param		tags	query		string	true	"Tag prefix to search"
//	@Success	200		{object}	map[string]interface{}
//	@Failure	400		{object}	map[string]interface{}
//	@Failure	404		{object}	map[string]interface{}
//	@Security	ApiKeyAuth
//	@Router		/api/xbooru/query_completion [get]
func (h *XbooruHandler) QueryCompletion(c fiber.Ctx) error {
	tags := c.Query("tags", "")
	if tags == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tags query parameter is required")
	}

	ctx := c.Context()
	res, err := h.Provider.QueryCompletion(ctx, tags)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("failed to fetch completion from Xbooru: %v", err))
	}

	if len(res) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"tags":    []string{},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"tags":    res,
	})
}

// ProxyMedia acts as a reverse proxy for media to bypass hotlinking.
//
//	@Summary	Proxy Xbooru Media
//	@Tags		xbooru
//	@Produce	*/*
//	@Param		url	query	string	true	"Direct media URL to proxy"
//	@Success	200	"Image or Video file"
//	@Failure	400	{string}	string	"Missing or invalid URL"
//	@Failure	500	{string}	string	"Failed to proxy media"
//	@Router		/api/xbooru/media [get]
func (h *XbooruHandler) ProxyMedia(c fiber.Ctx) error {
	targetURL := c.Query("url")
	if targetURL == "" {
		return c.Status(fiber.StatusBadRequest).SendString("URL parameter is required")
	}

	decodedURL, err := url.QueryUnescape(targetURL)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid URL format")
	}

	ctx := c.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, decodedURL, http.NoBody)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to create proxy request")
	}

	req.Header.Set("User-Agent", h.Provider.Cfg.UserAgent)
	req.Header.Set("Referer", h.Provider.Cfg.XbooruURL)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("Upstream fetch failed")
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).SendString("Upstream returned non-OK status")
	}

	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Set("Content-Length", resp.Header.Get("Content-Length"))
	c.Set("Cache-Control", "public, max-age=31536000") // 1 year cache

	// Synchronously copy the response body to the client to avoid premature close of resp.Body
	if _, err := io.Copy(c.Response().BodyWriter(), resp.Body); err != nil {
		return fiber.NewError(fiber.StatusBadGateway, "Failed to stream media content")
	}

	return nil
}

func (h *XbooruHandler) resolveMatoiURLs(baseURL string, posts []models.Post) {
	if h.Provider.Cfg.ResolverURL != "" {
		baseURL = h.Provider.Cfg.ResolverURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	for i := range posts {
		p := &posts[i]
		if p.FileURL != "" {
			p.MatoiFileURL = fmt.Sprintf("%s/api/xbooru/media?url=%s", baseURL, url.QueryEscape(p.FileURL))
		}
		if p.PreviewURL != "" {
			p.MatoiPreviewURL = fmt.Sprintf("%s/api/xbooru/media?url=%s", baseURL, url.QueryEscape(p.PreviewURL))
		}
		if p.SampleURL != "" {
			p.MatoiSampleURL = fmt.Sprintf("%s/api/xbooru/media?url=%s", baseURL, url.QueryEscape(p.SampleURL))
		}
	}
}

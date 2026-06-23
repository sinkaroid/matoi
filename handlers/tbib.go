package handlers

import (
	"crypto/tls"
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

// TbibHandler exposes the TBIB provider endpoints.
type TbibHandler struct {
	provider *providers.TbibProvider
}

// NewTbibHandler creates a new TBIB handler instance.
func NewTbibHandler(p *providers.TbibProvider) *TbibHandler {
	return &TbibHandler{provider: p}
}

// TbibResponse defines the JSON response schema for posts query.
type TbibResponse struct {
	Success  bool          `json:"success"`
	Provider string        `json:"provider"`
	Count    int           `json:"count"`
	Posts    []models.Post `json:"posts"`
}

// GetPosts returns posts from TBIB, checking Redis cache first.
//
//	@Summary	Get posts from TBIB
//	@Tags		tbib
//	@Produce	json
//	@Param		tags	query		string	false	"Space-separated tags"
//	@Param		limit	query		int		false	"Max results (default 20, max 100)"
//	@Param		page	query		int		false	"Page number (default 1)"
//	@Param		shuffle	query		bool	false	"Shuffle the results"
//	@Success	200		{object}	TbibResponse
//	@Failure	502		{object}	map[string]string	"Upstream fetch failed"
//	@Security	ApiKeyAuth
//	@Router		/api/tbib/posts [get]
//
//nolint:gocyclo,gocognit // Parsing request parameters makes this function complex
func (h *TbibHandler) GetPosts(c fiber.Ctx) error {
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

	// Cache key format: tbib:posts:tags_limit_page
	cacheKey := fmt.Sprintf("tbib:posts:%s:%d:%d", tags, limit, page)
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
			return c.Status(http.StatusNotFound).JSON(TbibResponse{
				Success:  false,
				Provider: "tbib",
				Count:    0,
				Posts:    []models.Post{},
			})
		}
		return c.Status(http.StatusOK).JSON(TbibResponse{
			Success:  true,
			Provider: "tbib",
			Count:    len(posts),
			Posts:    posts,
		})
	}

	// Fetch from upstream
	posts, err = h.provider.FetchPosts(c.Context(), tags, limit, page)
	if err != nil {
		return fiber.NewError(http.StatusBadGateway, fmt.Sprintf("Upstream fetch failed: %v", err))
	}

	// Only cache if we got results
	if len(posts) > 0 {
		ttl := h.provider.Cfg.RedisExpireCache
		if setErr := cache.Set(ctx, cacheKey, posts, ttl); setErr != nil {
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
		return c.Status(http.StatusNotFound).JSON(TbibResponse{
			Success:  false,
			Provider: "tbib",
			Count:    0,
			Posts:    []models.Post{},
		})
	}

	return c.Status(http.StatusOK).JSON(TbibResponse{
		Success:  true,
		Provider: "tbib",
		Count:    len(posts),
		Posts:    posts,
	})
}

// ProxyMedia fetches and streams blocked TBIB media.
//
//	@Summary	Proxy and stream TBIB media
//	@Tags		tbib
//	@Param		url	query	string	true	"Encoded TBIB media URL"
//	@Success	200	"Streams the media file"
//	@Failure	400	{object}	map[string]string	"Invalid parameters"
//	@Failure	502	{object}	map[string]string	"Failed to fetch media"
//	@Router		/api/tbib/media [get]
func (h *TbibHandler) ProxyMedia(c fiber.Ctx) error {
	mediaURL := c.Query("url", "")
	if mediaURL == "" {
		return fiber.NewError(http.StatusBadRequest, "Missing url parameter")
	}

	if !isValidTbibMediaDomain(mediaURL) {
		return fiber.NewError(http.StatusBadRequest, "Invalid media source domain")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
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

	if contentType := resp.Header.Get("Content-Type"); contentType != "" {
		c.Set("Content-Type", contentType)
	}
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		c.Set("Content-Length", contentLength)
	}

	if _, err := io.Copy(c.Response().BodyWriter(), resp.Body); err != nil {
		return fiber.NewError(http.StatusBadGateway, "Failed to stream media content")
	}

	return nil
}

func isValidTbibMediaDomain(mediaURL string) bool {
	parsedURL, err := url.Parse(mediaURL)
	if err != nil {
		return false
	}
	return strings.HasSuffix(parsedURL.Host, ".tbib.org") || parsedURL.Host == "tbib.org"
}

func (h *TbibHandler) resolveMatoiURLs(c fiber.Ctx, posts []models.Post) {
	baseURL := h.provider.Cfg.ResolverURL
	if baseURL == "" {
		baseURL = c.BaseURL()
	}

	baseURL = strings.TrimSuffix(baseURL, "/")

	for i := range posts {
		p := &posts[i]
		if p.FileURL != "" {
			p.MatoiFileURL = fmt.Sprintf("%s/api/tbib/media?url=%s", baseURL, url.QueryEscape(p.FileURL))
		}
		if p.PreviewURL != "" {
			p.MatoiPreviewURL = fmt.Sprintf("%s/api/tbib/media?url=%s", baseURL, url.QueryEscape(p.PreviewURL))
		}
		if p.SampleURL != "" {
			p.MatoiSampleURL = fmt.Sprintf("%s/api/tbib/media?url=%s", baseURL, url.QueryEscape(p.SampleURL))
		}
	}
}

// QueryCompletion handles the /api/tbib/query_completion endpoint.
//
//	@Summary		Get tag completion from TBIB (Eiyuu logic)
//	@Description	Scrapes autocomplete tags from TBIB using wildcard matching.
//	@Tags			tbib
//	@Produce		json
//	@Param			tags	query		string	true	"Tag query to autocomplete (e.g. yuri)"
//	@Success		200		{object}	QueryCompletionResponse
//	@Failure		400		{object}	main.ErrorResponse
//	@Failure		502		{object}	main.ErrorResponse
//	@Security		ApiKeyAuth
//	@Router			/api/tbib/query_completion [get]
func (h *TbibHandler) QueryCompletion(c fiber.Ctx) error {
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

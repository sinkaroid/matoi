package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
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

// FurbooruHandler handles HTTP requests for Furbooru.
type FurbooruHandler struct {
	Provider *providers.FurbooruProvider
}

// NewFurbooruHandler creates a new instance of FurbooruHandler.
func NewFurbooruHandler(provider *providers.FurbooruProvider) *FurbooruHandler {
	return &FurbooruHandler{
		Provider: provider,
	}
}

// FurbooruResponse defines the JSON response schema for posts query.
type FurbooruResponse struct {
	Success  bool          `json:"success"`
	Provider string        `json:"provider"`
	Count    int           `json:"count"`
	Posts    []models.Post `json:"posts"`
}

// GetPosts fetches posts from Furbooru.
//
//	@Summary		Fetch posts from Furbooru
//	@Description	Fetches a list of posts from Furbooru based on provided tags.
//	@Tags			furbooru
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			tags	query		string	false	"Tags to search for"
//	@Param			limit	query		int		false	"Max results (default 100)"
//	@Param			page	query		int		false	"Page number (default 1)"
//	@Param			shuffle	query		bool	false	"Shuffle the results"
//	@Success		200		{object}	FurbooruResponse
//	@Failure		500		{object}	map[string]string	"Internal server error"
//	@Router			/api/furbooru/posts [get]
//
//nolint:gocyclo,gocognit // Parsing request parameters makes this function complex
func (h *FurbooruHandler) GetPosts(c fiber.Ctx) error {
	tags := c.Query("tags", "")
	limitStr := c.Query("limit", "0")
	pageStr := c.Query("page", "1")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, "Invalid limit parameter")
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return fiber.NewError(http.StatusBadRequest, "Invalid page parameter")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("furbooru:posts:tags=%s:limit=%d:page=%d", urlQueryEncode(tags), limit, page)

	// Fast path: try cache
	var cachedPosts []models.Post
	found, err := cache.Get(ctx, cacheKey, &cachedPosts)
	if err != nil {
		log.Printf("Redis get error for furbooru posts: %v", err)
	}
	if found {
		c.Locals("source", "CACHE")
		h.resolveMatoiURLs(c, cachedPosts)

		if c.Query("shuffle", "false") == "true" && len(cachedPosts) > 0 {
			rand.Shuffle(len(cachedPosts), func(i, j int) {
				cachedPosts[i], cachedPosts[j] = cachedPosts[j], cachedPosts[i]
			})
		}

		if len(cachedPosts) == 0 {
			return c.Status(http.StatusNotFound).JSON(FurbooruResponse{
				Success:  false,
				Provider: "furbooru",
				Count:    0,
				Posts:    []models.Post{},
			})
		}

		return c.JSON(FurbooruResponse{
			Success:  true,
			Provider: "furbooru",
			Count:    len(cachedPosts),
			Posts:    cachedPosts,
		})
	}

	posts, err := h.Provider.FetchPosts(ctx, tags, limit, page)
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}

	// Cache the result asynchronously only if we have posts (do not cache empty 404s)
	if len(posts) > 0 {
		go func() {
			bgCtx, cancelBg := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelBg()
			if err := cache.Set(bgCtx, cacheKey, posts, h.Provider.Config.RedisExpireCache); err != nil {
				log.Printf("Failed to cache furbooru posts: %v", err)
			}
		}()
	}

	c.Locals("source", "FETCH")
	h.resolveMatoiURLs(c, posts)

	if c.Query("shuffle", "false") == "true" && len(posts) > 0 {
		rand.Shuffle(len(posts), func(i, j int) {
			posts[i], posts[j] = posts[j], posts[i]
		})
	}

	if len(posts) == 0 {
		return c.Status(http.StatusNotFound).JSON(FurbooruResponse{
			Success:  false,
			Provider: "furbooru",
			Count:    0,
			Posts:    []models.Post{},
		})
	}

	return c.JSON(FurbooruResponse{
		Success:  true,
		Provider: "furbooru",
		Count:    len(posts),
		Posts:    posts,
	})
}

// resolveMatoiURLs populates the Matoi proxy URLs for media.
func (h *FurbooruHandler) resolveMatoiURLs(c fiber.Ctx, posts []models.Post) {
	baseURL := h.Provider.Config.ResolverURL
	if baseURL == "" {
		baseURL = c.BaseURL()
	}

	// Ensure no trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	for i := range posts {
		p := &posts[i]
		if p.FileURL != "" {
			p.MatoiFileURL = fmt.Sprintf("%s/api/furbooru/media?url=%s", baseURL, url.QueryEscape(p.FileURL))
		}
		if p.PreviewURL != "" {
			p.MatoiPreviewURL = fmt.Sprintf("%s/api/furbooru/media?url=%s", baseURL, url.QueryEscape(p.PreviewURL))
		}
		if p.SampleURL != "" {
			p.MatoiSampleURL = fmt.Sprintf("%s/api/furbooru/media?url=%s", baseURL, url.QueryEscape(p.SampleURL))
		}
	}
}

// ProxyMedia fetches and streams blocked Furbooru media.
//
//	@Summary	Proxy and stream Furbooru media
//	@Tags		furbooru
//	@Param		url	query	string	true	"Encoded Furbooru media URL"
//	@Success	200	"Streams the media file"
//	@Failure	400	{object}	map[string]string	"Invalid parameters"
//	@Failure	502	{object}	map[string]string	"Failed to fetch media"
//	@Router		/api/furbooru/media [get]
//
//nolint:gocyclo // Proxy logic is inherently complex
func (h *FurbooruHandler) ProxyMedia(c fiber.Ctx) error {
	mediaURL := c.Query("url", "")
	if mediaURL == "" {
		return fiber.NewError(http.StatusBadRequest, "Missing url parameter")
	}

	parsedURL, err := url.Parse(mediaURL)
	if err != nil || !isValidFurbooruHost(parsedURL.Host) {
		return fiber.NewError(http.StatusBadRequest, "Invalid media source domain")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(c.Context(), http.MethodGet, mediaURL, http.NoBody)
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, "Failed to create proxy request")
	}

	userAgent := h.Provider.Config.UserAgent
	if userAgent == "" {
		userAgent = "Matoi/1.0"
	}
	req.Header.Set("User-Agent", userAgent)

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

// QueryCompletion provides tag autocomplete natively.
//
//	@Summary		Furbooru Tag Autocomplete
//	@Description	Provides tag completion via native tags search. NO CACHE.
//	@Tags			furbooru
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			tags	query		string	true	"Tag prefix to search for"
//	@Success		200		{object}	QueryCompletionResponse
//	@Failure		400		{object}	main.ErrorResponse
//	@Failure		500		{object}	main.ErrorResponse
//	@Router			/api/furbooru/query_completion [get]
func (h *FurbooruHandler) QueryCompletion(c fiber.Ctx) error {
	query := c.Query("tags", "")
	if query == "" {
		return fiber.NewError(http.StatusBadRequest, "tags query parameter is required")
	}

	// No cache for query completion
	tags, err := h.Provider.QueryCompletion(c.Context(), query)
	if err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}

	if len(tags) == 0 {
		return c.Status(http.StatusNotFound).JSON(QueryCompletionResponse{
			Success: false,
			Tags:    []string{},
		})
	}

	return c.JSON(QueryCompletionResponse{
		Success: true,
		Tags:    tags,
	})
}

func isValidFurbooruHost(host string) bool {
	return host == "furbooru.org" || strings.HasSuffix(host, ".furbooru.org") ||
		host == "furrycdn.org" || strings.HasSuffix(host, ".furrycdn.org")
}

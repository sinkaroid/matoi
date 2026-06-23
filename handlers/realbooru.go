package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"matoi/cache"
	"matoi/config"
	"matoi/models"
	"matoi/providers"

	"github.com/gofiber/fiber/v3"
)

// RealbooruHandler handles HTTP requests for Realbooru endpoints.
type RealbooruHandler struct {
	Provider *providers.RealbooruProvider
	Cfg      *config.Config
}

// NewRealbooruHandler creates a new handler instance.
func NewRealbooruHandler(p *providers.RealbooruProvider, cfg *config.Config) *RealbooruHandler {
	return &RealbooruHandler{
		Provider: p,
		Cfg:      cfg,
	}
}

// GetPosts fetches posts from Realbooru.
//
// @Summary Fetch posts from Realbooru
// @Description Fetches a list of posts from Realbooru using the provided tags, limit, and page.
// @Tags realbooru
// @Produce json
// @Param tags query string false "Space-separated list of tags to search for"
// @Param limit query int false "Number of posts to return (default is configured limit)"
// @Param page query int false "Page number (1-indexed, default is 1)"
// @Param shuffle query bool false "Shuffle the posts randomly"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/realbooru/posts [get]
//
//nolint:gocyclo // Caching and query parsing adds complexity naturally
func (h *RealbooruHandler) GetPosts(c fiber.Ctx) error {
	tags := c.Query("tags")
	limitStr := c.Query("limit")
	pageStr := c.Query("page")
	shuffleStr := c.Query("shuffle", "false")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 42
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	shuffle, err := strconv.ParseBool(shuffleStr)
	if err != nil {
		shuffle = false
	}

	cacheKey := fmt.Sprintf("realbooru:posts:tags=%s:limit=%d:page=%d", url.QueryEscape(tags), limit, page)

	var cachedPosts []models.Post
	found, err := cache.Get(c.Context(), cacheKey, &cachedPosts)
	if err != nil {
		log.Printf("Redis get error for realbooru posts: %v", err)
	}

	if found {
		c.Locals("source", "CACHE")
		if shuffle {
			rand.Shuffle(len(cachedPosts), func(i, j int) {
				cachedPosts[i], cachedPosts[j] = cachedPosts[j], cachedPosts[i]
			})
		}

		if len(cachedPosts) == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success":  false,
				"provider": "realbooru",
				"count":    0,
				"posts":    []models.Post{},
			})
		}

		cachedPosts = resolveRealbooruMatoiURLs(cachedPosts, c)

		return c.JSON(fiber.Map{
			"success":  true,
			"provider": "realbooru",
			"count":    len(cachedPosts),
			"posts":    cachedPosts,
		})
	}

	posts, err := h.Provider.FetchPosts(c.Context(), tags, limit, page)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if err := cache.Set(c.Context(), cacheKey, posts, h.Cfg.RedisExpireCache); err != nil {
		log.Printf("Redis set error for realbooru posts: %v", err)
	}

	c.Locals("source", "API")

	if shuffle {
		rand.Shuffle(len(posts), func(i, j int) {
			posts[i], posts[j] = posts[j], posts[i]
		})
	}

	// Always wrap empty arrays rather than null
	if len(posts) == 0 {
		posts = []models.Post{}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success":  false,
			"provider": "realbooru",
			"count":    0,
			"posts":    posts,
		})
	}

	posts = resolveRealbooruMatoiURLs(posts, c)

	return c.JSON(fiber.Map{
		"success":  true,
		"provider": "realbooru",
		"count":    len(posts),
		"posts":    posts,
	})
}

// QueryCompletion fetches Eiyuu tag autocompletions from Realbooru.
//
// @Summary Fetch tag autocompletions from Realbooru
// @Description Fetches tag autocomplete suggestions from Realbooru based on the query. NO CACHE.
// @Tags realbooru
// @Produce json
// @Param tags query string true "Tag query to autocomplete"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/realbooru/query_completion [get]
func (h *RealbooruHandler) QueryCompletion(c fiber.Ctx) error {
	query := c.Query("tags")
	if query == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tags parameter is required")
	}

	tags, err := h.Provider.QueryCompletion(c.Context(), query)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if len(tags) == 0 {
		tags = []string{}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"tags":    tags,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"tags":    tags,
	})
}

// ProxyMedia proxies an image URL to bypass Realbooru's hotlink protection.
//
// @Summary Proxy media to bypass Realbooru hotlink protection
// @Description Proxies a media URL through the server, attaching required referer headers.
// @Tags realbooru
// @Produce octet-stream
// @Param url query string true "URL of the media to proxy"
// @Success 200 {string} string "Media content"
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/realbooru/media [get]
//
//nolint:gocognit,gocyclo // Proxy logic naturally requires multiple checks
func (h *RealbooruHandler) ProxyMedia(c fiber.Ctx) error {
	targetURL := c.Query("url")
	if targetURL == "" {
		return fiber.NewError(fiber.StatusBadRequest, "url parameter is required")
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fiber.NewError(fiber.StatusBadRequest, "invalid url parameter")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	originalExt := path.Ext(parsedURL.Path)
	extensionsToTry := []string{originalExt, ".jpg", ".png", ".gif", ".jpeg", ".mp4", ".webm"}

	seen := make(map[string]bool)
	var uniqueExtensions []string
	for _, ext := range extensionsToTry {
		if ext == "" || seen[ext] {
			continue
		}
		seen[ext] = true
		uniqueExtensions = append(uniqueExtensions, ext)
	}

	var resp *http.Response
	var body io.ReadCloser

	for _, ext := range uniqueExtensions {
		var testURL string
		if originalExt != "" {
			testURL = targetURL[:strings.LastIndex(targetURL, originalExt)] + ext
		} else {
			testURL = targetURL + ext
		}

		req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, testURL, http.NoBody)
		if reqErr != nil {
			continue
		}

		req.Header.Set("Referer", "https://realbooru.com/")
		if h.Cfg.UserAgent != "" {
			req.Header.Set("User-Agent", h.Cfg.UserAgent)
		}

		r, doErr := client.Do(req)
		if doErr != nil {
			continue
		}

		if r.StatusCode == http.StatusOK {
			resp = r
			body = r.Body
			break
		}
		//nolint:errcheck // We intentionally ignore close errors on fast-fail
		_ = r.Body.Close()
	}

	if resp == nil {
		return fiber.NewError(fiber.StatusNotFound, "upstream returned status 404 for all common extensions")
	}
	defer func() {
		if body != nil {
			//nolint:errcheck // Ignore close error
			_ = body.Close()
		}
	}()

	c.Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Set("Cache-Control", "public, max-age=31536000")

	data, err := io.ReadAll(body)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to read upstream response")
	}

	return c.Send(data)
}

func resolveRealbooruMatoiURLs(posts []models.Post, c fiber.Ctx) []models.Post {
	baseURL := c.BaseURL()

	for i := range posts {
		if posts[i].FileURL != "" {
			posts[i].MatoiFileURL = fmt.Sprintf("%s/api/realbooru/media?url=%s", baseURL, url.QueryEscape(posts[i].FileURL))
		}
		if posts[i].PreviewURL != "" {
			posts[i].MatoiPreviewURL = fmt.Sprintf("%s/api/realbooru/media?url=%s", baseURL, url.QueryEscape(posts[i].PreviewURL))
		}
		if posts[i].SampleURL != "" {
			posts[i].MatoiSampleURL = fmt.Sprintf("%s/api/realbooru/media?url=%s", baseURL, url.QueryEscape(posts[i].SampleURL))
		}
	}
	return posts
}

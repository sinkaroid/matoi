// Package config handles application configuration and environment variables loading.
package config

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// AppVersion defines the current version of the application.
// Can be overridden at build time using -ldflags="-X 'matoi/config.AppVersion=xxx'"
var AppVersion = "10.0.0-alpha"

// Config holds all the configuration variables for the application.
type Config struct {
	Port                   string
	ResolverURL            string
	UserAgent              string
	EnableLogs             bool
	EnableGraphQL          bool
	APIKey                 string
	RedisURL               string
	RedisExpireCache       time.Duration
	DanbooruURL            string
	DanbooruReturnLmt      int
	DanbooruAPIID          string
	DanbooruAPIKey         string
	GelbooruURL            string
	GelbooruReturnLmt      int
	GelbooruAPIKey         string
	GelbooruUserID         string
	Rule34URL              string
	Rule34ReturnLimit      string
	Rule34APIID            string
	Rule34APIKey           string
	TbibURL                string
	TbibReturnLimit        string
	XbooruURL              string
	XbooruReturnLimit      string
	HypnohubURL            string
	HypnohubReturnLimit    string
	SafebooruURL           string
	SafebooruReturnLimit   string
	YandereURL             string
	YandereReturnLimit     string
	KonachanComURL         string
	KonachanComReturnLimit string
	KonachanNetURL         string
	KonachanNetReturnLimit string
	FlareSolverrURL        string
}

func getEnvWithDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

// LoadConfig loads the environment variables from .env or standard env variables, falling back to defaults if not set.
//
//nolint:gocyclo,gocognit // Config loading naturally has many assignments
func LoadConfig() *Config {
	// Loading .env is optional as environment variables can be set directly in production.
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file loaded, relying on environment variables.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	// Parse REDIS_EXPIRE_CACHE (in minutes, default to 5)
	expireStr := os.Getenv("REDIS_EXPIRE_CACHE")
	expireMinutes := 5
	if val, err := strconv.Atoi(strings.TrimSpace(expireStr)); err == nil && val > 0 {
		expireMinutes = val
	}
	redisExpire := time.Duration(expireMinutes) * time.Minute

	danbooruLimitVal := 100
	if val, err := strconv.Atoi(getEnvWithDefault("DANBOORU_RETURN_LIMIT", "100")); err == nil && val > 0 {
		danbooruLimitVal = val
	}

	gelbooruLimitVal := 100
	if val, err := strconv.Atoi(getEnvWithDefault("GELBOORU_RETURN_LIMIT", "100")); err == nil && val > 0 {
		gelbooruLimitVal = val
	}

	userAgent := os.Getenv("USER_AGENT")
	if userAgent == "" {
		// Provide a safe default if not set
		userAgent = fmt.Sprintf("matoi/%s %s", AppVersion, runtime.Version())
	}

	enableLogs := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_LOGS"))) == "true"
	enableGraphQL := strings.ToLower(strings.TrimSpace(os.Getenv("GRAPHQL"))) == "true"
	apiKey := os.Getenv("API_KEY")

	danbooruLimit := danbooruLimitVal
	gelbooruURL := getEnvWithDefault("GELBOORU_URL", "https://gelbooru.com/index.php")
	gelbooruLimit := gelbooruLimitVal
	rule34URL := getEnvWithDefault("RULE34_URL", "https://api.rule34.xxx/index.php")
	rule34Limit := getEnvWithDefault("RULE34_RETURN_LIMIT", "100")
	tbibURL := getEnvWithDefault("TBIB_URL", "https://tbib.org/index.php")
	tbibLimit := getEnvWithDefault("TBIB_RETURN_LIMIT", "100")
	xbooruURL := getEnvWithDefault("XBOORU_URL", "https://xbooru.com/index.php")
	xbooruLimit := getEnvWithDefault("XBOORU_RETURN_LIMIT", "100")
	hypnohubURL := getEnvWithDefault("HYPNOHUB_URL", "https://hypnohub.net/post.json")
	hypnohubLimit := getEnvWithDefault("HYPNOHUB_RETURN_LIMIT", "100")
	safebooruURL := getEnvWithDefault("SAFEBOORU_URL", "https://safebooru.org/index.php")
	safebooruLimit := getEnvWithDefault("SAFEBOORU_RETURN_LIMIT", "100")
	yandereURL := getEnvWithDefault("YANDERE_URL", "https://yande.re/post.json")
	yandereLimit := getEnvWithDefault("YANDERE_RETURN_LIMIT", "100")
	konachanComURL := getEnvWithDefault("KONACHANCOM_URL", "https://konachan.com/post.json")
	konachanComLimit := getEnvWithDefault("KONACHANCOM_RETURN_LIMIT", "100")
	konachanNetURL := getEnvWithDefault("KONACHANNET_URL", "https://konachan.net/post.json")
	konachanNetLimit := getEnvWithDefault("KONACHANNET_RETURN_LIMIT", "100")

	flareSolverrURL := os.Getenv("FLARESOLVERR_URL")

	return &Config{
		Port:                   port,
		ResolverURL:            os.Getenv("RESOLVER_URL"),
		UserAgent:              userAgent,
		EnableLogs:             enableLogs,
		EnableGraphQL:          enableGraphQL,
		APIKey:                 apiKey,
		RedisURL:               redisURL,
		RedisExpireCache:       redisExpire,
		DanbooruURL:            os.Getenv("DANBOORU_URL"),
		DanbooruReturnLmt:      danbooruLimit,
		DanbooruAPIID:          os.Getenv("DANBOORU_API_ID"),
		DanbooruAPIKey:         os.Getenv("DANBOORU_API_KEY"),
		GelbooruURL:            gelbooruURL,
		GelbooruReturnLmt:      gelbooruLimit,
		GelbooruAPIKey:         os.Getenv("GELBOORU_API_KEY"),
		GelbooruUserID:         os.Getenv("GELBOORU_API_ID"),
		Rule34URL:              rule34URL,
		Rule34ReturnLimit:      rule34Limit,
		Rule34APIID:            os.Getenv("RULE34_API_ID"),
		Rule34APIKey:           os.Getenv("RULE34_API_KEY"),
		TbibURL:                tbibURL,
		TbibReturnLimit:        tbibLimit,
		XbooruURL:              xbooruURL,
		XbooruReturnLimit:      xbooruLimit,
		HypnohubURL:            hypnohubURL,
		HypnohubReturnLimit:    hypnohubLimit,
		SafebooruURL:           safebooruURL,
		SafebooruReturnLimit:   safebooruLimit,
		YandereURL:             yandereURL,
		YandereReturnLimit:     yandereLimit,
		KonachanComURL:         konachanComURL,
		KonachanComReturnLimit: konachanComLimit,
		KonachanNetURL:         konachanNetURL,
		KonachanNetReturnLimit: konachanNetLimit,
		FlareSolverrURL:        flareSolverrURL,
	}
}

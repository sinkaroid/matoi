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
var AppVersion = "2.0.1-alpha"

// Config holds all the configuration variables for the application.
type Config struct {
	Port              string
	ResolverURL       string
	UserAgent         string
	EnableLogs        bool
	APIKey            string
	RedisURL          string
	RedisExpireCache  time.Duration
	DanbooruURL       string
	DanbooruReturnLmt int
	DanbooruAPIID     string
	DanbooruAPIKey    string
	GelbooruURL       string
	GelbooruReturnLmt int
	GelbooruAPIKey    string
	GelbooruUserID    string
	Rule34URL         string
	Rule34ReturnLimit string
	Rule34APIID       string
	Rule34APIKey      string
}

// LoadConfig loads the environment variables from .env or standard env variables, falling back to defaults if not set.
//
//nolint:gocyclo // Config loading naturally has many assignments
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

	rule34URL := os.Getenv("RULE34_URL")
	if rule34URL == "" {
		rule34URL = "https://api.rule34.xxx/index.php"
	}

	rule34Limit := os.Getenv("RULE34_RETURN_LIMIT")
	if rule34Limit == "" {
		rule34Limit = "100"
	}

	gelbooruURL := os.Getenv("GELBOORU_URL")
	if gelbooruURL == "" {
		gelbooruURL = "https://gelbooru.com/index.php"
	}

	gelbooruLimit := 100
	if val, err := strconv.Atoi(os.Getenv("GELBOORU_RETURN_LIMIT")); err == nil && val > 0 {
		gelbooruLimit = val
	}

	userAgent := os.Getenv("USER_AGENT")
	if userAgent == "" {
		// Provide a safe default if not set
		userAgent = fmt.Sprintf("matoi/%s %s", AppVersion, runtime.Version())
	}

	enableLogs := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_LOGS"))) == "true"
	apiKey := os.Getenv("API_KEY")

	danbooruLimit := 100
	if val, err := strconv.Atoi(os.Getenv("DANBOORU_RETURN_LIMIT")); err == nil && val > 0 {
		danbooruLimit = val
	}

	return &Config{
		Port:              port,
		ResolverURL:       os.Getenv("RESOLVER_URL"),
		UserAgent:         userAgent,
		EnableLogs:        enableLogs,
		APIKey:            apiKey,
		RedisURL:          redisURL,
		RedisExpireCache:  redisExpire,
		DanbooruURL:       os.Getenv("DANBOORU_URL"),
		DanbooruReturnLmt: danbooruLimit,
		DanbooruAPIID:     os.Getenv("DANBOORU_API_ID"),
		DanbooruAPIKey:    os.Getenv("DANBOORU_API_KEY"),
		GelbooruURL:       gelbooruURL,
		GelbooruReturnLmt: gelbooruLimit,
		GelbooruAPIKey:    os.Getenv("GELBOORU_API_KEY"),
		GelbooruUserID:    os.Getenv("GELBOORU_API_ID"),
		Rule34URL:         rule34URL,
		Rule34ReturnLimit: rule34Limit,
		Rule34APIID:       os.Getenv("RULE34_API_ID"),
		Rule34APIKey:      os.Getenv("RULE34_API_KEY"),
	}
}

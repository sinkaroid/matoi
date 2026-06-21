// Package config handles application configuration and environment variables loading.
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all the configuration variables for the application.
type Config struct {
	Port              string
	ResolverURL       string
	UserAgent         string
	EnableLogs        bool
	RedisURL          string
	RedisExpireCache  time.Duration
	DanbooruAPIKey    string
	DanbooruLogin     string
	GelbooruAPIKey    string
	GelbooruUserID    string
	Rule34URL         string
	Rule34ReturnLimit string
	Rule34APIID       string
	Rule34APIKey      string
}

// LoadConfig loads the environment variables from .env or standard env variables, falling back to defaults if not set.
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

	userAgent := os.Getenv("USER_AGENT")
	if userAgent == "" {
		// Provide a safe default if not set
		userAgent = "matoi/1.0.2-alpha Go/1.26.4"
	}

	enableLogs := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_LOGS"))) == "true"

	return &Config{
		Port:              port,
		ResolverURL:       os.Getenv("RESOLVER_URL"),
		UserAgent:         userAgent,
		EnableLogs:        enableLogs,
		RedisURL:          redisURL,
		RedisExpireCache:  redisExpire,
		DanbooruAPIKey:    os.Getenv("DANBOORU_API_KEY"),
		DanbooruLogin:     os.Getenv("DANBOORU_LOGIN"),
		GelbooruAPIKey:    os.Getenv("GELBOORU_API_KEY"),
		GelbooruUserID:    os.Getenv("GELBOORU_USER_ID"),
		Rule34URL:         rule34URL,
		Rule34ReturnLimit: rule34Limit,
		Rule34APIID:       os.Getenv("RULE34_API_ID"),
		Rule34APIKey:      os.Getenv("RULE34_API_KEY"),
	}
}

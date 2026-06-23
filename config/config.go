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
var AppVersion = "15.6.1-alpha"

// Config holds all the configuration variables for the application.
type Config struct {
	Port                string
	ResolverURL         string
	UserAgent           string
	EnableLogs          bool
	EnableGraphQL       bool
	APIKey              string
	RedisURL            string
	RedisExpireCache    time.Duration
	PostRestReturnLimit int
	E621APIID           string
	E621APIKey          string
	E926APIID           string
	E926APIKey          string
	FurbooruAPIKey      string
	DerpibooruAPIKey    string
	DanbooruAPIID       string
	DanbooruAPIKey      string
	GelbooruAPIKey      string
	GelbooruUserID      string
	Rule34APIID         string
	Rule34APIKey        string
	FlareSolverrURL     string
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

	port := os.Getenv("MATOI_PORT")
	if port == "" {
		port = "3000"
	}

	redisURL := os.Getenv("MATOI_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	// Parse MATOI_REDIS_EXPIRE_CACHE (in minutes, default to 5)
	expireStr := os.Getenv("MATOI_REDIS_EXPIRE_CACHE")
	expireMinutes := 5
	if val, err := strconv.Atoi(strings.TrimSpace(expireStr)); err == nil && val > 0 {
		expireMinutes = val
	}
	redisExpire := time.Duration(expireMinutes) * time.Minute

	postRestReturnLimit := 100
	if val, err := strconv.Atoi(getEnvWithDefault("MATOI_POST_REST_RETURN_LIMIT", "100")); err == nil && val > 0 {
		postRestReturnLimit = val
	}

	userAgent := os.Getenv("MATOI_USER_AGENT")
	if userAgent == "" {
		// Provide a safe default if not set
		userAgent = fmt.Sprintf("matoi/%s %s", AppVersion, runtime.Version())
	}

	enableLogs := strings.ToLower(strings.TrimSpace(os.Getenv("MATOI_ENABLE_LOGS"))) == "true"
	enableGraphQL := strings.ToLower(strings.TrimSpace(os.Getenv("MATOI_GRAPHQL"))) == "true"
	apiKey := os.Getenv("MATOI_API_KEY")

	return &Config{
		Port:                port,
		ResolverURL:         os.Getenv("MATOI_RESOLVER_URL"),
		UserAgent:           userAgent,
		EnableLogs:          enableLogs,
		EnableGraphQL:       enableGraphQL,
		APIKey:              apiKey,
		RedisURL:            redisURL,
		RedisExpireCache:    redisExpire,
		PostRestReturnLimit: postRestReturnLimit,
		E621APIID:           os.Getenv("E621_API_ID"),
		E621APIKey:          os.Getenv("E621_API_KEY"),
		E926APIID:           os.Getenv("E926_API_ID"),
		E926APIKey:          os.Getenv("E926_API_KEY"),
		FurbooruAPIKey:      os.Getenv("FURBOORU_API_KEY"),
		DerpibooruAPIKey:    os.Getenv("DERPIBOORU_API_KEY"),
		DanbooruAPIID:       os.Getenv("DANBOORU_API_ID"),
		DanbooruAPIKey:      os.Getenv("DANBOORU_API_KEY"),
		GelbooruAPIKey:      os.Getenv("GELBOORU_API_KEY"),
		GelbooruUserID:      os.Getenv("GELBOORU_API_ID"),
		Rule34APIID:         os.Getenv("RULE34_API_ID"),
		Rule34APIKey:        os.Getenv("RULE34_API_KEY"),
		FlareSolverrURL:     os.Getenv("MATOI_FLARESOLVERR_URL"),
	}
}

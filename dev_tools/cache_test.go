package dev_tools

import (
	"context"
	"matoi/config"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestFlushCache(t *testing.T) {
	// Need to load from parent directory because test runs in dev_tools
	cfg := config.LoadConfig()
	if cfg.RedisURL == "" {
		// Fallback if .env wasn't loaded due to working directory
		cfg.RedisURL = "redis://localhost:6379/0"
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		t.Fatalf("Error parsing redis url: %v", err)
	}
	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("Error flushing redis: %v", err)
	}
	t.Log("Redis Cache Flushed Successfully!")
}

package dev_tools

import (
	"context"
	"testing"
	"time"

	"matoi/config"
	prov "matoi/providers"

	"github.com/joho/godotenv"
)

func TestKonachanNetDirect(t *testing.T) {
	// Load .env file from project root
	err := godotenv.Load("../.env")
	if err != nil {
		t.Logf("Warning: Could not load .env file: %v", err)
	}

	cfg := config.LoadConfig()

	// Print FlareSolverr config to verify
	if cfg.FlareSolverrURL != "" {
		t.Logf("FlareSolverr is configured: %s", cfg.FlareSolverrURL)
	} else {
		t.Log("FlareSolverr is NOT configured")
	}

	p := prov.NewKonachanNetProvider(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()

	tags := "yuri"
	limit := 3
	page := 1

	t.Logf("Fetching posts from KonachanNet with tags=%s, limit=%d, page=%d...", tags, limit, page)
	posts, err := p.FetchPosts(ctx, tags, limit, page)
	if err != nil {
		t.Fatalf("Failed to fetch posts: %v", err)
	}

	if len(posts) == 0 {
		t.Fatalf("No posts returned")
	}

	t.Logf("Success! Fetched %d posts.", len(posts))
	for i, post := range posts {
		t.Logf("Post %d: ID=%d, URL=%s", i+1, post.ID, post.FileURL)
	}

	// Test Query Completion
	query := "yur"
	t.Logf("Testing Query Completion for '%s'...", query)
	qc, err := p.QueryCompletion(ctx, query)
	if err != nil {
		t.Fatalf("Failed query completion: %v", err)
	}

	if len(qc) > 0 {
		t.Logf("Success! Query completion returned %d tags (First 3: %v)", len(qc), qc[:min(3, len(qc))])
	} else {
		t.Log("Query completion returned 0 tags.")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

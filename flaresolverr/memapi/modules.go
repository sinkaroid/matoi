package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var cachedModules []string

func init() {
	go fetchModules()
}

func fetchModules() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://raw.githubusercontent.com/FlareSolverr/FlareSolverr/refs/heads/master/requirements.txt", http.NoBody)
	if err != nil {
		cachedModules = []string{"error: failed to create request"}
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		cachedModules = []string{"error: failed to fetch"}
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("close error: %v\n", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		cachedModules = []string{"error: failed to read"}
		return
	}

	lines := strings.Split(string(body), "\n")
	var mods []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			mods = append(mods, line)
		}
	}
	cachedModules = mods
}

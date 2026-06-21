// Package handlers contains all the HTTP endpoints handlers for the application.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v3"
)

var startTime = time.Now()

type ipWhoisResponse struct {
	Success bool   `json:"success"`
	Country string `json:"country"`
	Region  string `json:"region"`
	City    string `json:"city"`
}

// HomeResponse represents the structured JSON response for the root endpoint,
// preserving the exact key ordering during serialization.
type HomeResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Endpoint string `json:"endpoint"`
	Date     string `json:"date"`
	RSS      string `json:"rss"`
	Heap     string `json:"heap"`
	Server   string `json:"server"`
	Version  string `json:"version"`
	Uptime   string `json:"uptime"`
}

// HomeHandler responds with system info, server location, and uptime.
//
//	@Summary	System info and health check
//	@Tags		system
//	@Produce	json
//	@Success	200	{object}	HomeResponse
//	@Router		/ [get]
func HomeHandler(c fiber.Ctx) error {
	// Geolocation fetch from ipwho.is using Fiber's context
	serverLocation := getGeolocation(c.Context())

	// Memory Stats
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	rssStr := fmt.Sprintf("%.1f MB", float64(mem.Sys)/1024/1024)
	heapStr := fmt.Sprintf("%.2f/%.2f MB", float64(mem.HeapSys)/1024/1024, float64(mem.HeapAlloc)/1024/1024)

	// Uptime calculation
	uptimeStr := time.Since(startTime).Truncate(time.Second).String()

	// Format Date (equivalent to JS toLocaleString)
	dateStr := time.Now().Format("1/2/2006, 3:04:05 PM")

	return c.Status(fiber.StatusOK).JSON(HomeResponse{
		Success:  true,
		Message:  "Hi, I'm alive!",
		Endpoint: "https://github.com/sinkaroid/matoi/blob/master/README.md#routing",
		Date:     dateStr,
		RSS:      rssStr,
		Heap:     heapStr,
		Server:   serverLocation,
		Version:  "1.0.0",
		Uptime:   uptimeStr,
	})
}

func getGeolocation(ctx context.Context) string {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ipwho.is", http.NoBody)
	if err != nil {
		return "Unknown"
	}

	resp, err := client.Do(req)
	if err != nil {
		return "Unknown"
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			_ = closeErr // satisfy staticcheck by using the error variable without empty branch
		}
	}()

	var geo ipWhoisResponse
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return "Unknown"
	}

	if !geo.Success {
		return "Unknown"
	}

	if geo.Region != "" {
		return geo.Country + ", " + geo.Region
	}
	if geo.City != "" {
		return geo.Country + ", " + geo.City
	}
	return geo.Country
}

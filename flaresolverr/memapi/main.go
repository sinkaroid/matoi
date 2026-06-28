// Package main provides a lightweight memory API for FlareSolverr.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var startTime = time.Now()

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.URL.Path != "/memory" && r.URL.Path != "/" && r.URL.Path != "/metrics" {
		http.NotFound(w, r)
		return
	}

	memBytes, source := getMemory()
	memLimit := getLimit()

	if r.URL.Path == "/metrics" {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		metrics := fmt.Sprintf("# HELP flaresolverr_memory_bytes Total memory usage in bytes (RSS of flaresolverr and its descendants)\n"+
			"# TYPE flaresolverr_memory_bytes gauge\n"+
			"flaresolverr_memory_bytes %d\n"+
			"# HELP flaresolverr_memory_limit_bytes Memory limit in bytes\n"+
			"# TYPE flaresolverr_memory_limit_bytes gauge\n"+
			"flaresolverr_memory_limit_bytes %d\n"+
			"# HELP flaresolverr_uptime_seconds Uptime of the sidecar in seconds\n"+
			"# TYPE flaresolverr_uptime_seconds counter\n"+
			"flaresolverr_uptime_seconds %f\n", memBytes, memLimit, time.Since(startTime).Seconds())

		if _, err := w.Write([]byte(metrics)); err != nil {
			fmt.Printf("write error: %v\n", err)
		}
		return
	}

	memMB := float64(memBytes) / (1024 * 1024)
	limitMB := float64(memLimit) / (1024 * 1024)

	rssStr := fmt.Sprintf("%.2f MB", memMB)
	heapStr := fmt.Sprintf("%.2f/Unknown MB", memMB)
	if memLimit > 0 {
		heapStr = fmt.Sprintf("%.2f/%.2f MB", memMB, limitMB)
	}

	uptimeStr := time.Since(startTime).Round(time.Second).String()

	resp := MemoryResponse{
		Success:      true,
		MemoryMB:     memMB,
		MemoryBytes:  memBytes,
		CgroupSource: source,
		RSS:          rssStr,
		Heap:         heapStr,
		Uptime:       uptimeStr,
		Modules:      getCachedModules(),
	}

	w.Header().Set("Content-Type", "application/json")
	if encodeErr := json.NewEncoder(w).Encode(resp); encodeErr != nil {
		http.Error(w, encodeErr.Error(), http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8192", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}

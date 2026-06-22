package main

// MemoryResponse defines the JSON structure for the memory API.
type MemoryResponse struct {
	Success      bool     `json:"success"`
	MemoryMB     float64  `json:"memory_mb"`
	MemoryBytes  int64    `json:"memory_bytes"`
	CgroupSource string   `json:"cgroup_source"`
	RSS          string   `json:"rss"`
	Heap         string   `json:"heap"`
	Uptime       string   `json:"uptime"`
	Modules      []string `json:"modules"`
}

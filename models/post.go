// Package models defines the normalized shared structures used throughout the application.
package models

// Post represents a normalized post across all booru providers.
type Post struct {
	Provider        string   `json:"provider"`
	ID              int      `json:"id"`
	Directory       int      `json:"directory"`
	FileURL         string   `json:"file_url"`
	PreviewURL      string   `json:"preview_url"`
	SampleURL       string   `json:"sample_url"`
	MatoiFileURL    string   `json:"matoi_file_url"`
	MatoiPreviewURL string   `json:"matoi_preview_url"`
	MatoiSampleURL  string   `json:"matoi_sample_url"`
	Rating          string   `json:"rating"` // s (safe), q (questionable), e (explicit)
	Score           int      `json:"score"`
	Source          string   `json:"source"`
	Image           string   `json:"image"`
	Tags            []string `json:"tags"`
}

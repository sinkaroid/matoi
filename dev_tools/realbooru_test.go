package dev_tools

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestRealbooruScraping(t *testing.T) {
	urlStr := "https://realbooru.com/index.php?page=post&s=list&tags=sfw&pid=0"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, http.NoBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Matoi/14.0")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		t.Fatalf("Failed to parse HTML: %v", err)
	}

	doc.Find("div.col.thumb").Each(func(i int, s *goquery.Selection) {
		idStr, _ := s.Attr("id") // Example: id="s997493"
		idStr = strings.TrimPrefix(idStr, "s")

		img := s.Find("img")
		previewURL, _ := img.Attr("src")
		tagsStr, _ := img.Attr("title")

		// Split tags by comma, trim, and replace spaces with underscores to match canonical tag format
		var tagsArray []string
		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tag = strings.ReplaceAll(tag, " ", "_")
					tagsArray = append(tagsArray, tag)
				}
			}
		}

		// Infer FileURL based on previewURL
		// Preview: https://realbooru.com/thumbnails/96/1b/thumbnail_961bb8b41346899b60c72a2f0d14aee5.jpg
		// Full: https://realbooru.com/images/96/1b/961bb8b41346899b60c72a2f0d14aee5.jpg
		fileURL := strings.Replace(previewURL, "thumbnails/", "images/", 1)
		fileURL = strings.Replace(fileURL, "thumbnail_", "", 1)

		directory := ""
		imageName := ""
		if parsed, pErr := url.Parse(fileURL); pErr == nil {
			parts := strings.Split(parsed.Path, "/")
			if len(parts) >= 4 {
				imageName = parts[len(parts)-1]
				directory = parts[len(parts)-3] + "/" + parts[len(parts)-2]
			} else if len(parts) >= 3 {
				imageName = parts[len(parts)-1]
				directory = parts[len(parts)-2]
			}
		}

		fmt.Printf("ID: %s\n", idStr)
		fmt.Printf("Directory: %s\n", directory)
		fmt.Printf("Image: %s\n", imageName)
		fmt.Printf("Preview: %s\n", previewURL)
		fmt.Printf("File (Guessed): %s\n", fileURL)
		fmt.Printf("Tags: %v\n", tagsArray)
		fmt.Println("--------------------------------------------------")
	})
}

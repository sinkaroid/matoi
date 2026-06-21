package dev_tools_test

import (
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestGoquery(t *testing.T) {
	query := "jeanne"
	urlStr := fmt.Sprintf("https://rule34.xxx/index.php?page=tags&s=list&tags=*%s*&sort=desc&order_by=index_count", query)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("User-Agent", "matoi/dev")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		t.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	var tags []string
	doc.Find("table.highlightable a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, "&tags=") {
			parts := strings.Split(href, "&tags=")
			if len(parts) > 1 {
				decodedURL, _ := url.QueryUnescape(parts[1])
				decodedHTML := html.UnescapeString(decodedURL)
				tags = append(tags, decodedHTML)
			}
		}
	})

	t.Log("Result Tags:")
	for _, tag := range tags {
		t.Log("-", tag)
	}
}

package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
)

func main() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", "https://api.rule34.xxx/index.php?page=dapi&s=post&q=index&json=1&limit=2", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Matoi/1.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Body length: %d\n", len(body))
	if len(body) > 500 {
		fmt.Printf("Body: %s\n", string(body[:500]))
	} else {
		fmt.Printf("Body: %s\n", string(body))
	}
}

package search

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func Search(query string) ([]SearchResult, error) {
	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request build error: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; infra-search/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	var results []SearchResult
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if i >= 10 {
			return
		}
		titleEl := s.Find(".result__a")
		title := strings.TrimSpace(titleEl.Text())
		href, _ := titleEl.Attr("href")

		// DuckDuckGo はリダイレクト URL を返すので uddg パラメータから実 URL を取得
		parsedHref, err := url.Parse(href)
		actualURL := href
		if err == nil {
			if uddg := parsedHref.Query().Get("uddg"); uddg != "" {
				actualURL = uddg
			}
		}

		snippet := strings.TrimSpace(s.Find(".result__snippet").Text())
		source := extractDomain(actualURL)

		if title != "" && actualURL != "" {
			results = append(results, SearchResult{
				Title:   title,
				URL:     actualURL,
				Snippet: snippet,
				Source:  source,
			})
		}
	})

	return results, nil
}

func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := parsed.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}

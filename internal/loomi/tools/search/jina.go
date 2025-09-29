package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// JinaClient handles Jina AI search functionality
type JinaClient struct {
	token     string
	searchURL string
	client    *http.Client
}

// NewJinaClient creates a new Jina search client
func NewJinaClient() *JinaClient {
	token := os.Getenv("JINA_API_KEY")
	searchURL := os.Getenv("JINA_SEARCH_URL")
	if searchURL == "" {
		searchURL = "https://s.jina.ai/"
	}

	return &JinaClient{
		token:     token,
		searchURL: searchURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchRequest represents a search request payload
type SearchRequest struct {
	Query string `json:"q"`
}

// SearchResult represents the search result structure
type SearchResult struct {
	Platform      string `json:"platform"`
	Keyword       string `json:"keyword"`
	SearchResults struct {
		Content string `json:"content"`
		Source  string `json:"source"`
	} `json:"search_results"`
	Success bool   `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SearchWeb performs web search using Jina AI
func (j *JinaClient) SearchWeb(ctx context.Context, query string) (map[string]any, error) {
	payload := SearchRequest{Query: query}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", j.searchURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+j.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Respond-With", "no-content")

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("search failed with status code: %d", resp.StatusCode),
		}, nil
	}

	// Read response body
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If JSON decode fails, treat as plain text
		body := make([]byte, 0)
		resp.Body.Read(body)
		result = map[string]any{
			"platform": "web",
			"keyword":  query,
			"search_results": map[string]any{
				"content": string(body),
				"source":  "jina_ai",
			},
		}
	}

	return result, nil
}

// SearchWebSimpleFormat performs web search and returns simplified format
func (j *JinaClient) SearchWebSimpleFormat(ctx context.Context, query string) (map[string]any, error) {
	fullResult, err := j.SearchWeb(ctx, query)
	if err != nil {
		return map[string]any{"posts": []any{}}, err
	}

	// If search failed, return empty posts
	if success, ok := fullResult["success"].(bool); ok && !success {
		return map[string]any{"posts": []any{}}, nil
	}

	// Extract content and parse into posts format
	if searchResults, ok := fullResult["search_results"].(map[string]any); ok {
		if content, ok := searchResults["content"].(string); ok {
			posts := j.parseContentToPosts(content)
			return map[string]any{
				"posts":    posts,
				"platform": "web",
				"keyword":  query,
			}, nil
		}
	}

	return map[string]any{"posts": []any{}}, nil
}

// parseContentToPosts parses search content into posts format
func (j *JinaClient) parseContentToPosts(content string) []map[string]any {
	// Simple parsing logic - extract URLs and content snippets
	// This is a simplified version, real implementation would be more sophisticated
	lines := strings.Split(content, "\n")
	posts := make([]map[string]any, 0)

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Extract URLs from line
		url := j.extractURL(line)
		if url == "" {
			url = fmt.Sprintf("https://example.com/result-%d", i+1)
		}

		post := map[string]any{
			"title":       j.truncateString(line, 100),
			"content":     line,
			"url":         url,
			"source":      "jina_ai",
			"platform":    "web",
			"description": j.truncateString(line, 200),
		}

		posts = append(posts, post)

		// Limit to 5 posts
		if len(posts) >= 5 {
			break
		}
	}

	return posts
}

// extractURL extracts URL from text (simplified)
func (j *JinaClient) extractURL(text string) string {
	// Look for http/https URLs
	words := strings.Fields(text)
	for _, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			return word
		}
	}
	return ""
}

// truncateString truncates string to specified length
func (j *JinaClient) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

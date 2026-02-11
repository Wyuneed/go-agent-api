package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/wyuneed/go-agent-api/internal/application/usecase/tool"
	"github.com/wyuneed/go-agent-api/pkg/toolspec"
)

type WebSearchTool struct {
	tool.BaseTool
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewWebSearchTool(apiKey string) *WebSearchTool {
	return &WebSearchTool{
		apiKey:  apiKey,
		baseURL: "https://api.search.brave.com/res/v1/web/search",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (t *WebSearchTool) Name() string {
	return "web_search"
}

func (t *WebSearchTool) Description() string {
	return "Search the web for current information. Use this when you need up-to-date information, facts you're unsure about, or to verify claims."
}

func (t *WebSearchTool) Definition() toolspec.Tool {
	return toolspec.NewTool(t.Name(), t.Description(), &toolspec.JSONSchema{
		Type: "object",
		Properties: map[string]toolspec.PropertySchema{
			"query": {
				Type:        "string",
				Description: "The search query. Be specific and use relevant keywords.",
			},
			"num_results": {
				Type:        "integer",
				Description: "Number of results to return (1-10). Default is 5.",
				Default:     5,
			},
		},
		Required: []string{"query"},
	})
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	numResults := 5
	if n, ok := args["num_results"].(float64); ok && n >= 1 && n <= 10 {
		numResults = int(n)
	}

	u, _ := url.Parse(t.baseURL)
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", numResults))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Subscription-Token", t.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return t.formatResults(result), nil
}

func (t *WebSearchTool) formatResults(raw map[string]any) map[string]any {
	results := []map[string]string{}

	if web, ok := raw["web"].(map[string]any); ok {
		if items, ok := web["results"].([]any); ok {
			for _, item := range items {
				if r, ok := item.(map[string]any); ok {
					results = append(results, map[string]string{
						"title":       getString(r, "title"),
						"url":         getString(r, "url"),
						"description": getString(r, "description"),
					})
				}
			}
		}
	}

	return map[string]any{
		"results": results,
		"count":   len(results),
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

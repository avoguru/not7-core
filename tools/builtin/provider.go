package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/not7/core/tools"
)

// Provider implements built-in tools with direct HTTP calls
type Provider struct {
	serpAPIKey string
	httpClient *http.Client
}

// NewProvider creates a new builtin tool provider
func NewProvider(serpAPIKey string) *Provider {
	return &Provider{
		serpAPIKey: serpAPIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Initialize sets up the provider
func (p *Provider) Initialize(config map[string]string) error {
	if apiKey, ok := config["serp_api_key"]; ok && apiKey != "" {
		p.serpAPIKey = apiKey
	}

	if p.serpAPIKey == "" {
		return fmt.Errorf("SERP API key is required for builtin web search")
	}

	return nil
}

// ListTools returns available built-in tools
func (p *Provider) ListTools(ctx context.Context) ([]tools.ToolDefinition, error) {
	return []tools.ToolDefinition{
		{
			Name:        "WebSearch",
			Description: "Search the web using Google Search. Returns titles, URLs, and snippets of search results.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
					"num_results": map[string]interface{}{
						"type":        "integer",
						"description": "Number of results to return (default: 5)",
					},
				},
				"required": []string{"query"},
			},
			Provider: "builtin",
		},
		{
			Name:        "WebFetch",
			Description: "Fetch and extract text content from a URL. Returns the main text content of the webpage.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to fetch",
					},
				},
				"required": []string{"url"},
			},
			Provider: "builtin",
		},
	}, nil
}

// ExecuteTool executes a built-in tool
func (p *Provider) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*tools.ToolResult, error) {
	switch toolName {
	case "WebSearch":
		return p.executeWebSearch(ctx, arguments)
	case "WebFetch":
		return p.executeWebFetch(ctx, arguments)
	default:
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", toolName),
		}, nil
	}
}

// executeWebSearch performs web search via SerpAPI
func (p *Provider) executeWebSearch(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &tools.ToolResult{
			Success: false,
			Error:   "query parameter is required",
		}, nil
	}

	numResults := 5
	if num, ok := args["num_results"].(float64); ok {
		numResults = int(num)
	}

	// Build SerpAPI request
	apiURL := fmt.Sprintf("https://serpapi.com/search?q=%s&api_key=%s&num=%d",
		url.QueryEscape(query),
		p.serpAPIKey,
		numResults,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("search request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("SerpAPI error (status %d): %s", resp.StatusCode, string(body)),
		}, nil
	}

	var serpResp struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&serpResp); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to decode response: %v", err),
		}, nil
	}

	// Format results
	var results []map[string]string
	for _, result := range serpResp.OrganicResults {
		results = append(results, map[string]string{
			"title":   result.Title,
			"url":     result.Link,
			"snippet": result.Snippet,
		})
	}

	return &tools.ToolResult{
		Success: true,
		Output:  results,
	}, nil
}

// executeWebFetch fetches and extracts text from a URL
func (p *Provider) executeWebFetch(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	urlStr, ok := args["url"].(string)
	if !ok || urlStr == "" {
		return &tools.ToolResult{
			Success: false,
			Error:   "url parameter is required",
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create request: %v", err),
		}, nil
	}

	req.Header.Set("User-Agent", "NOT7-Agent/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("fetch request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("HTTP error: status %d", resp.StatusCode),
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read response: %v", err),
		}, nil
	}

	// Simple text extraction - remove HTML tags
	text := extractText(string(body))

	// Limit to reasonable size (first 5000 chars)
	if len(text) > 5000 {
		text = text[:5000] + "...\n[Content truncated]"
	}

	return &tools.ToolResult{
		Success: true,
		Output:  text,
	}, nil
}

// extractText performs basic HTML tag removal
func extractText(html string) string {
	// Remove script and style tags with content
	html = removeTagContent(html, "script")
	html = removeTagContent(html, "style")

	// Remove HTML tags
	var result strings.Builder
	inTag := false
	for _, char := range html {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(char)
		}
	}

	// Clean up whitespace
	text := result.String()
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Collapse multiple newlines
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// removeTagContent removes HTML tags and their content
func removeTagContent(html, tagName string) string {
	startTag := "<" + tagName
	endTag := "</" + tagName + ">"

	for {
		start := strings.Index(strings.ToLower(html), strings.ToLower(startTag))
		if start == -1 {
			break
		}

		end := strings.Index(strings.ToLower(html[start:]), strings.ToLower(endTag))
		if end == -1 {
			break
		}

		html = html[:start] + html[start+end+len(endTag):]
	}

	return html
}

// GetProviderName returns the provider identifier
func (p *Provider) GetProviderName() string {
	return "builtin"
}

// Close cleans up resources
func (p *Provider) Close() error {
	return nil
}

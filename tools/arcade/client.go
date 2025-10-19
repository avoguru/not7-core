package arcade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL       = "https://api.arcade.dev"
	cacheDuration = 1 * time.Hour
)

// Client handles Arcade API interactions
type Client struct {
	apiKey     string
	userID     string
	httpClient *http.Client

	// Simple cache for tools
	cachedTools   []Tool
	cacheExpiry   time.Time
}

// NewClient creates a new Arcade API client
func NewClient(apiKey, userID string) *Client {
	return &Client{
		apiKey: apiKey,
		userID: userID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListTools returns all available tools for a given toolkit (with 1-hour cache)
func (c *Client) ListTools(toolkit string) ([]Tool, error) {
	// Check cache
	if time.Now().Before(c.cacheExpiry) && len(c.cachedTools) > 0 {
		return c.cachedTools, nil
	}

	// Fetch from API
	url := fmt.Sprintf("%s/v1/tools?toolkit=%s&limit=100", baseURL, toolkit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var toolsResp ToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&toolsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Update cache
	c.cachedTools = toolsResp.Items
	c.cacheExpiry = time.Now().Add(cacheDuration)

	return toolsResp.Items, nil
}

// ExecuteTool executes an Arcade tool
func (c *Client) ExecuteTool(toolName string, inputs map[string]interface{}) (interface{}, error) {
	url := fmt.Sprintf("%s/v1/tools/execute", baseURL)

	reqBody := ExecuteToolRequest{
		ToolName: toolName,
		Input:    inputs,
		UserID:   c.userID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute tool: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle authorization errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("authorization required: please run './not7 authorize arcade' first")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var execResp ExecuteToolResponse
	if err := json.Unmarshal(body, &execResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for top-level errors
	if execResp.Error != "" {
		return nil, fmt.Errorf("tool execution error: %s", execResp.Error)
	}

	// Check for failure status
	if !execResp.Success || execResp.Status != "success" {
		if execResp.Output.Error != nil {
			return nil, fmt.Errorf("tool execution failed: %v", execResp.Output.Error)
		}
		return nil, fmt.Errorf("tool execution failed with status: %s", execResp.Status)
	}

	return execResp.Output.Value, nil
}

// AuthorizeTool initiates OAuth authorization for Gmail
func (c *Client) AuthorizeTool(toolName string) (*AuthorizationResponse, error) {
	url := fmt.Sprintf("%s/v1/tools/authorize", baseURL)

	reqBody := AuthorizeToolRequest{
		ToolName: toolName,
		UserID:   c.userID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize tool: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var authResp AuthorizationResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &authResp, nil
}

// CheckAuthStatus checks the status of an authorization with long-polling support
func (c *Client) CheckAuthStatus(authID string, waitSeconds int) (*AuthStatusResponse, error) {
	url := fmt.Sprintf("%s/v1/auth/status?id=%s", baseURL, authID)
	if waitSeconds > 0 && waitSeconds <= 59 {
		url += fmt.Sprintf("&wait=%d", waitSeconds)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var statusResp AuthStatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &statusResp, nil
}

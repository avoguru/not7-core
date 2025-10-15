package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NOT7Client is an HTTP client for the NOT7 API
type NOT7Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new NOT7 API client
func NewClient(baseURL string) *NOT7Client {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &NOT7Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Long timeout for sync operations
		},
	}
}

// RunAgent executes an agent (sync or async)
func (c *NOT7Client) RunAgent(agentJSON []byte, async bool) (map[string]interface{}, error) {
	url := c.baseURL + "/api/v1/agents/run"
	if async {
		url += "?async=true"
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(agentJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("server error: %s", result["error"])
	}

	return result, nil
}

// GetExecutionStatus gets the status of an execution
func (c *NOT7Client) GetExecutionStatus(execID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/executions/%s/status", c.baseURL, execID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("execution not found or error: %s", result["error"])
	}

	return result, nil
}

// GetExecutionResult gets the final result of an execution
func (c *NOT7Client) GetExecutionResult(execID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/executions/%s/result", c.baseURL, execID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("result not available: %s", result["error"])
	}

	return result, nil
}

// ListAgents lists all deployed agents
func (c *NOT7Client) ListAgents() (map[string]interface{}, error) {
	url := c.baseURL + "/api/v1/agents"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// CheckHealth checks if server is healthy
func (c *NOT7Client) CheckHealth() error {
	url := c.baseURL + "/health"

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("server not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("server unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

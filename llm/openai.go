package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/not7/core/spec"
)

// OpenAIClient handles communication with OpenAI API
type OpenAIClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient() (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	return &OpenAIClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// CompletionRequest represents OpenAI API request
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse represents OpenAI API response
type CompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Execute runs an LLM completion
func (c *OpenAIClient) Execute(config *spec.LLMConfig, prompt string, input string) (string, float64, error) {
	// Build request
	req := CompletionRequest{
		Model: config.Model,
		Messages: []Message{
			{Role: "system", Content: prompt},
		},
		Temperature: config.Temperature,
	}

	// Add user input if provided
	if input != "" {
		req.Messages = append(req.Messages, Message{
			Role:    "user",
			Content: input,
		})
	}

	// Set max tokens if specified
	if config.MaxTokens > 0 {
		req.MaxTokens = config.MaxTokens
	}

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var completion CompletionResponse
	if err := json.Unmarshal(body, &completion); err != nil {
		return "", 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", 0, fmt.Errorf("no completion choices returned")
	}

	// Calculate cost (approximate)
	cost := calculateCost(config.Model, completion.Usage)

	return completion.Choices[0].Message.Content, cost, nil
}

// calculateCost estimates the cost based on token usage
func calculateCost(model string, usage Usage) float64 {
	// Approximate pricing (as of 2024)
	var inputCostPer1k, outputCostPer1k float64

	switch model {
	case "gpt-4":
		inputCostPer1k = 0.03
		outputCostPer1k = 0.06
	case "gpt-4-turbo", "gpt-4-turbo-preview":
		inputCostPer1k = 0.01
		outputCostPer1k = 0.03
	case "gpt-3.5-turbo":
		inputCostPer1k = 0.0005
		outputCostPer1k = 0.0015
	default:
		// Default to gpt-3.5-turbo pricing
		inputCostPer1k = 0.0005
		outputCostPer1k = 0.0015
	}

	inputCost := float64(usage.PromptTokens) / 1000.0 * inputCostPer1k
	outputCost := float64(usage.CompletionTokens) / 1000.0 * outputCostPer1k

	return inputCost + outputCost
}


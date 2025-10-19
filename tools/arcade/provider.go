package arcade

import (
	"context"
	"fmt"
	"time"

	"github.com/not7/core/tools"
)

// Provider implements Arcade tool provider
type Provider struct {
	client      *Client
	toolkit     string                   // The Arcade toolkit to use (e.g., "Gmail", "Spotify", "Slack")
	toolNameMap map[string]string        // Maps short name (e.g., "SendEmail") to fully qualified name (e.g., "Gmail.SendEmail@3.2.1")
}

// NewProvider creates a new Arcade provider for a specific toolkit
func NewProvider(apiKey, userID, toolkit string) *Provider {
	return &Provider{
		client:      NewClient(apiKey, userID),
		toolkit:     toolkit,
		toolNameMap: make(map[string]string),
	}
}

// Initialize sets up the provider with configuration
func (p *Provider) Initialize(config map[string]string) error {
	// Config is already set via constructor
	// This method is for runtime config updates if needed
	if apiKey, ok := config["arcade_api_key"]; ok && apiKey != "" {
		p.client.apiKey = apiKey
	}
	if userID, ok := config["arcade_user_id"]; ok && userID != "" {
		p.client.userID = userID
	}

	if p.client.apiKey == "" {
		return fmt.Errorf("ARCADE_API_KEY is required")
	}
	if p.client.userID == "" {
		return fmt.Errorf("ARCADE_USER_ID is required")
	}

	return nil
}

// CheckAndHandleAuthorization checks if tools are authorized and handles interactive auth if needed
func (p *Provider) CheckAndHandleAuthorization(ctx context.Context) error {
	// Get tools for this toolkit to check authorization status
	tools, err := p.client.ListTools(p.toolkit)
	if err != nil {
		return fmt.Errorf("failed to check authorization status: %w", err)
	}

	if len(tools) == 0 {
		return fmt.Errorf("no %s tools available", p.toolkit)
	}

	// Check if already authorized
	if tools[0].Requirements.Authorization != nil && tools[0].Requirements.Authorization.Status == "active" {
		return nil // Already authorized
	}

	// Need authorization - initiate OAuth flow
	return p.interactiveAuthorize(ctx, tools[0].FullyQualifiedName)
}

// interactiveAuthorize handles the interactive OAuth authorization flow
func (p *Provider) interactiveAuthorize(ctx context.Context, toolName string) error {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("  🔐 %s Authorization Required\n", p.toolkit)
	fmt.Println()
	fmt.Printf("  This agent requires access to %s.\n", p.toolkit)
	fmt.Println("  Please authorize to continue...")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Initiate authorization
	authResp, err := p.client.AuthorizeTool(toolName)
	if err != nil {
		return fmt.Errorf("failed to initiate authorization: %w", err)
	}

	if authResp.Status == "completed" {
		fmt.Println("✅ Already authorized!")
		return nil
	}

	if authResp.AuthorizationURL == "" {
		return fmt.Errorf("no authorization URL received")
	}

	// Display authorization URL
	fmt.Println("📋 Authorization URL:")
	fmt.Println()
	fmt.Println("  " + authResp.AuthorizationURL)
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("⏳ Waiting for authorization (timeout: 5 minutes)...")
	fmt.Println("   Press Ctrl+C to cancel")
	fmt.Println()

	// Poll for authorization completion
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("authorization timeout (5 minutes)")
		case <-ctx.Done():
			return fmt.Errorf("authorization cancelled")
		case <-ticker.C:
			// Check status with long-polling (30 seconds)
			statusResp, err := p.client.CheckAuthStatus(authResp.AuthorizationID, 30)
			if err != nil {
				// Continue trying on errors
				continue
			}

			if statusResp.Status == "completed" {
				fmt.Println("✅ Authorization completed!")
				fmt.Println()
				fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
				fmt.Println()
				fmt.Println("🚀 Continuing with agent execution...")
				fmt.Println()
				return nil
			}

			if statusResp.Status == "failed" {
				errorMsg := "unknown error"
				if statusResp.Error != "" {
					errorMsg = statusResp.Error
				}
				return fmt.Errorf("authorization failed: %s", errorMsg)
			}
		}
	}
}

// ListTools returns all available tools for this toolkit from Arcade
func (p *Provider) ListTools(ctx context.Context) ([]tools.ToolDefinition, error) {
	arcadeTools, err := p.client.ListTools(p.toolkit)
	if err != nil {
		return nil, fmt.Errorf("failed to list Arcade %s tools: %w", p.toolkit, err)
	}

	// Convert Arcade tools to ToolDefinition format
	toolDefs := make([]tools.ToolDefinition, 0, len(arcadeTools))
	for _, arcadeTool := range arcadeTools {
		// Build input schema from parameters
		inputSchema := map[string]interface{}{
			"type":       "object",
			"properties": make(map[string]interface{}),
			"required":   make([]string, 0),
		}

		properties := inputSchema["properties"].(map[string]interface{})
		required := inputSchema["required"].([]string)

		for _, param := range arcadeTool.Input.Parameters {
			if name, ok := param["name"].(string); ok {
				properties[name] = param
				if isRequired, ok := param["required"].(bool); ok && isRequired {
					required = append(required, name)
				}
			}
		}
		inputSchema["required"] = required

		// Store mapping from short name to fully qualified name
		p.toolNameMap[arcadeTool.Name] = arcadeTool.FullyQualifiedName

		toolDef := tools.ToolDefinition{
			Name:        arcadeTool.Name,
			Description: arcadeTool.Description,
			InputSchema: inputSchema,
			Provider:    "arcade",
		}
		toolDefs = append(toolDefs, toolDef)
	}

	return toolDefs, nil
}

// ExecuteTool executes an Arcade tool
func (p *Provider) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*tools.ToolResult, error) {
	// Map short name to fully qualified name
	fullyQualifiedName, ok := p.toolNameMap[toolName]
	if !ok {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", toolName),
		}, nil
	}

	// Execute tool via Arcade API using fully qualified name
	output, err := p.client.ExecuteTool(fullyQualifiedName, arguments)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Output:  output,
	}, nil
}

// GetProviderName returns the provider identifier
func (p *Provider) GetProviderName() string {
	return "arcade"
}

// Close cleans up resources
func (p *Provider) Close() error {
	// No cleanup needed for REST API client
	return nil
}

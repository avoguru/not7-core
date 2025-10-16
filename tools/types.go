package tools

import "context"

// ToolDefinition represents a tool with its schema
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema,omitempty"`
	Provider    string                 `json:"provider"` // "builtin", "mcp", etc.
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success bool                   `json:"success"`
	Output  interface{}            `json:"output"`
	Error   string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool invocation request
type ToolCall struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolProvider is the interface that all tool providers must implement
// This abstraction allows us to support multiple backends (builtin, MCP, custom)
type ToolProvider interface {
	// Initialize the provider with configuration
	Initialize(config map[string]string) error

	// ListTools returns all available tools from this provider
	ListTools(ctx context.Context) ([]ToolDefinition, error)

	// ExecuteTool executes a tool with given arguments
	ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error)

	// GetProviderName returns the provider identifier
	GetProviderName() string

	// Close cleans up resources
	Close() error
}

// ToolConfig represents tool configuration in agent spec
type ToolConfig struct {
	Provider string            `json:"provider"` // "builtin" or "mcp"
	Config   map[string]string `json:"config"`   // Provider-specific config
	Enabled  []string          `json:"enabled"`  // List of enabled tool names (optional, empty = all)
}

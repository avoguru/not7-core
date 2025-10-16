package tools

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager coordinates tool providers and execution
type Manager struct {
	providers map[string]ToolProvider
	registry  *Registry
	mu        sync.RWMutex
	userID    string // User ID for tool execution
}

// NewManager creates a new tool manager
func NewManager(userID string) *Manager {
	return &Manager{
		providers: make(map[string]ToolProvider),
		registry:  NewRegistry(),
		userID:    userID,
	}
}

// RegisterProvider adds a tool provider and loads its tools
func (m *Manager) RegisterProvider(provider ToolProvider) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	providerName := provider.GetProviderName()
	if _, exists := m.providers[providerName]; exists {
		return fmt.Errorf("provider already registered: %s", providerName)
	}

	// Load tools from provider
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tools, err := provider.ListTools(ctx)
	if err != nil {
		return NewProviderError(providerName, "failed to list tools", err)
	}

	// Register each tool
	for _, tool := range tools {
		if err := m.registry.Register(tool); err != nil {
			return NewProviderError(providerName, fmt.Sprintf("failed to register tool %s", tool.Name), err)
		}
	}

	m.providers[providerName] = provider
	return nil
}

// ExecuteTool executes a tool by name
func (m *Manager) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	// Get tool definition
	toolDef, err := m.registry.Get(toolName)
	if err != nil {
		return nil, NewToolError(toolName, "tool not found in registry", err)
	}

	// Get provider
	m.mu.RLock()
	provider, exists := m.providers[toolDef.Provider]
	m.mu.RUnlock()

	if !exists {
		return nil, NewToolError(toolName, fmt.Sprintf("provider not found: %s", toolDef.Provider), nil)
	}

	// Execute tool
	result, err := provider.ExecuteTool(ctx, toolName, arguments)
	if err != nil {
		return nil, NewToolError(toolName, "execution failed", err)
	}

	return result, nil
}

// ListTools returns all available tools
func (m *Manager) ListTools() []ToolDefinition {
	return m.registry.List()
}

// GetToolContext returns formatted tool descriptions for LLM
func (m *Manager) GetToolContext() string {
	return m.registry.GetToolContext()
}

// HasTools returns true if any tools are registered
func (m *Manager) HasTools() bool {
	return m.registry.Count() > 0
}

// Close shuts down all providers
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, provider := range m.providers {
		if err := provider.Close(); err != nil {
			errs = append(errs, NewProviderError(name, "failed to close", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing providers: %v", errs)
	}

	return nil
}

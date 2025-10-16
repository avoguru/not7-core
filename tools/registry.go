package tools

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry stores and manages available tools
type Registry struct {
	tools map[string]*ToolDefinition
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*ToolDefinition),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool ToolDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	r.tools[tool.Name] = &tool
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (*ToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool, nil
}

// List returns all registered tools
func (r *Registry) List() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, *tool)
	}

	// Sort by name for consistent ordering
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools
}

// Clear removes all tools from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]*ToolDefinition)
}

// Count returns the number of registered tools
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// GetToolContext generates a formatted string of available tools for LLM context
func (r *Registry) GetToolContext() string {
	tools := r.List()
	if len(tools) == 0 {
		return "No tools available."
	}

	var sb strings.Builder
	sb.WriteString("Available Tools:\n\n")

	for i, tool := range tools {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, tool.Name))
		sb.WriteString(fmt.Sprintf("   Description: %s\n", tool.Description))

		if tool.InputSchema != nil && len(tool.InputSchema) > 0 {
			sb.WriteString("   Parameters:\n")
			for key, val := range tool.InputSchema {
				sb.WriteString(fmt.Sprintf("     - %s: %v\n", key, val))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

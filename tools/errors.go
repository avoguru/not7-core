package tools

import "fmt"

// ToolError represents errors during tool execution
type ToolError struct {
	ToolName string
	Message  string
	Cause    error
}

func (e *ToolError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool '%s' failed: %s (cause: %v)", e.ToolName, e.Message, e.Cause)
	}
	return fmt.Sprintf("tool '%s' failed: %s", e.ToolName, e.Message)
}

func (e *ToolError) Unwrap() error {
	return e.Cause
}

// NewToolError creates a new tool error
func NewToolError(toolName, message string, cause error) *ToolError {
	return &ToolError{
		ToolName: toolName,
		Message:  message,
		Cause:    cause,
	}
}

// ProviderError represents errors from tool providers
type ProviderError struct {
	Provider string
	Message  string
	Cause    error
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("provider '%s' error: %s (cause: %v)", e.Provider, e.Message, e.Cause)
	}
	return fmt.Sprintf("provider '%s' error: %s", e.Provider, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// NewProviderError creates a new provider error
func NewProviderError(provider, message string, cause error) *ProviderError {
	return &ProviderError{
		Provider: provider,
		Message:  message,
		Cause:    cause,
	}
}

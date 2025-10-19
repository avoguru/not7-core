package arcade

// Tool represents an Arcade tool definition
type Tool struct {
	FullyQualifiedName string `json:"fully_qualified_name"`
	QualifiedName      string `json:"qualified_name"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Input              struct {
		Parameters []map[string]interface{} `json:"parameters"`
	} `json:"input"`
	Requirements struct {
		Met           bool `json:"met"`
		Authorization *struct {
			ID           string `json:"id"`
			ProviderID   string `json:"provider_id"`
			ProviderType string `json:"provider_type"`
			Status       string `json:"status"`
		} `json:"authorization,omitempty"`
	} `json:"requirements"`
}

// AuthRequirement defines OAuth requirements for a tool
type AuthRequirement struct {
	ProviderID string   `json:"provider_id"`
	Scopes     []string `json:"scopes,omitempty"`
}

// ToolsResponse is the response from GET /v1/tools
type ToolsResponse struct {
	Items      []Tool `json:"items"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	PageCount  int    `json:"page_count"`
	TotalCount int    `json:"total_count"`
}

// ExecuteToolRequest is the request to execute a tool
type ExecuteToolRequest struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	UserID   string                 `json:"user_id"`
}

// ExecuteToolResponse is the response from tool execution
type ExecuteToolResponse struct {
	ID            string  `json:"id"`
	ExecutionID   string  `json:"execution_id"`
	ExecutionType string  `json:"execution_type"`
	FinishedAt    string  `json:"finished_at"`
	Duration      float64 `json:"duration"`
	Status        string  `json:"status"` // "success", "failed", etc.
	Output        struct {
		Value interface{} `json:"value"`
		Error interface{} `json:"error,omitempty"`
	} `json:"output"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"` // Top-level error if request failed
}

// AuthorizeToolRequest initiates authorization for a tool
type AuthorizeToolRequest struct {
	ToolName string `json:"tool_name"`
	UserID   string `json:"user_id"`
}

// AuthorizationResponse contains authorization details
type AuthorizationResponse struct {
	AuthorizationID  string `json:"authorization_id"`
	Status           string `json:"status"` // "pending", "completed", "failed"
	AuthorizationURL string `json:"authorization_url,omitempty"`
	Error            string `json:"error,omitempty"`
}

// AuthStatusResponse is the response from GET /v1/auth/status
type AuthStatusResponse struct {
	AuthorizationID string `json:"authorization_id"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
}

package server

// ExecutionResponse represents the API response for agent execution
type ExecutionResponse struct {
	ID       string  `json:"id"`
	Status   string  `json:"status"`
	Goal     string  `json:"goal,omitempty"`
	Output   string  `json:"output,omitempty"`
	Error    string  `json:"error,omitempty"`
	Cost     float64 `json:"cost,omitempty"`
	Duration int64   `json:"duration_ms,omitempty"`
	LogFile  string  `json:"log_file,omitempty"`
}

// AgentInfo represents agent metadata
type AgentInfo struct {
	ID        string `json:"id"`
	Goal      string `json:"goal"`
	CreatedAt string `json:"created_at"`
}

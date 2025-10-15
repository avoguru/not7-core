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

// ExecutionStatus represents the current state of an execution
type ExecutionStatus struct {
	ExecutionID string    `json:"execution_id"`
	Status      string    `json:"status"` // queued, running, completed, failed
	AgentID     string    `json:"agent_id,omitempty"`
	Goal        string    `json:"goal"`
	StartedAt   string    `json:"started_at,omitempty"`
	CompletedAt string    `json:"completed_at,omitempty"`
	Progress    *Progress `json:"progress,omitempty"`
	CostSoFar   float64   `json:"cost_so_far,omitempty"`
	ElapsedMs   int64     `json:"elapsed_ms,omitempty"`
}

// Progress tracks execution progress
type Progress struct {
	TotalNodes      int    `json:"total_nodes"`
	CompletedNodes  int    `json:"completed_nodes"`
	CurrentNode     string `json:"current_node,omitempty"`
	CurrentNodeType string `json:"current_node_type,omitempty"`
}

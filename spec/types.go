package spec

// AgentSpec represents the complete NOT7 agent specification
type AgentSpec struct {
	ID       string         `json:"id,omitempty"`
	Version  string         `json:"version"`
	Goal     string         `json:"goal"`
	Config   *Config        `json:"config,omitempty"`
	Nodes    []Node         `json:"nodes"`
	Routes   []Route        `json:"routes"`
	Metadata *Metadata      `json:"metadata,omitempty"`
}

// Config holds global configuration
type Config struct {
	LLM         *LLMConfig    `json:"llm,omitempty"`
	Constraints *Constraints  `json:"constraints,omitempty"`
}

// LLMConfig defines language model settings
type LLMConfig struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

// Constraints define execution limits
type Constraints struct {
	MaxTime    string  `json:"max_time,omitempty"`
	MaxCost    float64 `json:"max_cost,omitempty"`
	MaxRetries int     `json:"max_retries,omitempty"`
}

// Node represents a single execution unit
type Node struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Type         string     `json:"type"` // "llm", "react", "tool", "transform", "conditional"
	Prompt       string     `json:"prompt,omitempty"`
	InputFormat  string     `json:"input_format,omitempty"`
	OutputFormat string     `json:"output_format,omitempty"`
	LLM          *LLMConfig `json:"llm,omitempty"`
	
	// ReAct-specific fields
	ReActGoal      string `json:"react_goal,omitempty"`
	MaxIterations  int    `json:"max_iterations,omitempty"`
	ThinkingPrompt string `json:"thinking_prompt,omitempty"`
}

// Route defines connection between nodes
type Route struct {
	From      string     `json:"from"`
	To        string     `json:"to"`
	Condition *Condition `json:"condition,omitempty"`
	Parallel  bool       `json:"parallel,omitempty"`
}

// Condition defines routing logic
type Condition struct {
	Type       string `json:"type"`       // "success", "failure", "expression"
	Expression string `json:"expression,omitempty"`
}

// Metadata holds execution results
type Metadata struct {
	CreatedAt       string        `json:"created_at,omitempty"`
	ExecutedAt      string        `json:"executed_at,omitempty"`
	ExecutionTimeMs int64         `json:"execution_time_ms,omitempty"`
	TotalCost       float64       `json:"total_cost,omitempty"`
	Status          string        `json:"status,omitempty"`
	NodeResults     []NodeResult  `json:"node_results,omitempty"`
}

// NodeResult holds results from a single node execution
type NodeResult struct {
	NodeID          string      `json:"node_id"`
	Status          string      `json:"status"`
	ExecutionTimeMs int64       `json:"execution_time_ms"`
	Cost            float64     `json:"cost,omitempty"`
	Input           interface{} `json:"input,omitempty"`
	Output          interface{} `json:"output,omitempty"`
	Error           string      `json:"error,omitempty"`
	ReActTrace      *ReActTrace `json:"react_trace,omitempty"`
}

// ReActTrace holds iteration details for ReAct nodes
type ReActTrace struct {
	Iterations          int            `json:"iterations"`
	ThinkingSteps       []ThinkingStep `json:"thinking_steps"`
	TotalThinkingTimeMs int64          `json:"total_thinking_time_ms"`
	IterationsCost      float64        `json:"iterations_cost"`
}

// ThinkingStep represents one iteration of ReAct thinking
type ThinkingStep struct {
	Iteration  int     `json:"iteration"`
	Thought    string  `json:"thought"`
	DurationMs int64   `json:"duration_ms"`
	Cost       float64 `json:"cost"`
}

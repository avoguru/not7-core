package execution

import (
	"time"

	"github.com/not7/core/spec"
)

// Execution represents a single agent execution instance
type Execution struct {
	ID        string           `json:"id"`
	Spec      *spec.AgentSpec  `json:"spec"`
	Status    Status           `json:"status"`
	Result    *Result          `json:"result,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	StartedAt *time.Time       `json:"started_at,omitempty"`
	EndedAt   *time.Time       `json:"ended_at,omitempty"`
}

// Status represents the current state of an execution
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Result contains the output and metadata from an execution
type Result struct {
	Output       string             `json:"output"`
	Error        string             `json:"error,omitempty"`
	DurationMs   int64              `json:"duration_ms"`
	TotalCost    float64            `json:"total_cost"`
	Metadata     *spec.Metadata     `json:"metadata,omitempty"`
}

// Options configures how an execution should be performed
type Options struct {
	// Stream enables live streaming of agent reasoning
	Stream bool

	// Async runs the execution in the background
	Async bool

	// Timeout sets the maximum execution duration (0 = no timeout)
	Timeout time.Duration
}

// ExecutionInfo is a lightweight summary of an execution
type ExecutionInfo struct {
	ID        string    `json:"id"`
	Goal      string    `json:"goal"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	DurationMs int64    `json:"duration_ms,omitempty"`
	TotalCost float64   `json:"total_cost,omitempty"`
}

// NewExecution creates a new execution instance
func NewExecution(id string, agentSpec *spec.AgentSpec) *Execution {
	return &Execution{
		ID:        id,
		Spec:      agentSpec,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}

// MarkStarted transitions execution to running state
func (e *Execution) MarkStarted() {
	now := time.Now()
	e.StartedAt = &now
	e.Status = StatusRunning
}

// MarkCompleted transitions execution to completed state with result
func (e *Execution) MarkCompleted(result *Result) {
	now := time.Now()
	e.EndedAt = &now
	e.Status = StatusCompleted
	e.Result = result
}

// MarkFailed transitions execution to failed state with error
func (e *Execution) MarkFailed(err error) {
	now := time.Now()
	e.EndedAt = &now
	e.Status = StatusFailed
	e.Result = &Result{
		Error: err.Error(),
	}
}

// Info returns a lightweight summary
func (e *Execution) Info() *ExecutionInfo {
	info := &ExecutionInfo{
		ID:        e.ID,
		Goal:      e.Spec.Goal,
		Status:    e.Status,
		CreatedAt: e.CreatedAt,
	}

	if e.Result != nil {
		info.DurationMs = e.Result.DurationMs
		info.TotalCost = e.Result.TotalCost
	}

	return info
}

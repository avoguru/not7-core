package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/not7/core/spec"
)

// ExecutionManager tracks running and completed executions
type ExecutionManager struct {
	executions map[string]*ExecutionStatus
	mu         sync.RWMutex
	execDir    string
}

// NewExecutionManager creates a new execution manager
func NewExecutionManager(execDir string) *ExecutionManager {
	return &ExecutionManager{
		executions: make(map[string]*ExecutionStatus),
		execDir:    execDir,
	}
}

// CreateExecution creates a new execution record
func (em *ExecutionManager) CreateExecution(execID string, agentSpec *spec.AgentSpec) *ExecutionStatus {
	em.mu.Lock()
	defer em.mu.Unlock()

	status := &ExecutionStatus{
		ExecutionID: execID,
		Status:      "queued",
		AgentID:     agentSpec.ID,
		Goal:        agentSpec.Goal,
		StartedAt:   time.Now().Format(time.RFC3339),
		Progress: &Progress{
			TotalNodes:     len(agentSpec.Nodes),
			CompletedNodes: 0,
		},
	}

	em.executions[execID] = status
	em.saveStatus(status)
	return status
}

// UpdateStatus updates execution status
func (em *ExecutionManager) UpdateStatus(execID string, status string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if exec, ok := em.executions[execID]; ok {
		exec.Status = status
		if status == "completed" || status == "failed" {
			exec.CompletedAt = time.Now().Format(time.RFC3339)
		}
		em.saveStatus(exec)
	}
}

// UpdateProgress updates execution progress
func (em *ExecutionManager) UpdateProgress(execID string, completed int, currentNode, nodeType string, costSoFar float64, elapsedMs int64) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if exec, ok := em.executions[execID]; ok {
		exec.Progress.CompletedNodes = completed
		exec.Progress.CurrentNode = currentNode
		exec.Progress.CurrentNodeType = nodeType
		exec.CostSoFar = costSoFar
		exec.ElapsedMs = elapsedMs
		em.saveStatus(exec)
	}
}

// GetStatus returns execution status
func (em *ExecutionManager) GetStatus(execID string) (*ExecutionStatus, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if status, ok := em.executions[execID]; ok {
		return status, nil
	}

	// Try loading from file
	return em.loadStatus(execID)
}

// ListExecutions returns all executions
func (em *ExecutionManager) ListExecutions() []*ExecutionStatus {
	em.mu.RLock()
	defer em.mu.RUnlock()

	executions := make([]*ExecutionStatus, 0, len(em.executions))
	for _, exec := range em.executions {
		executions = append(executions, exec)
	}
	return executions
}

// saveStatus saves execution status to file
func (em *ExecutionManager) saveStatus(status *ExecutionStatus) error {
	statusPath := filepath.Join(em.execDir, status.ExecutionID+".status.json")
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statusPath, data, 0644)
}

// loadStatus loads execution status from file
func (em *ExecutionManager) loadStatus(execID string) (*ExecutionStatus, error) {
	statusPath := filepath.Join(em.execDir, execID+".status.json")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return nil, err
	}

	var status ExecutionStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// SaveResult saves execution result to file
func (em *ExecutionManager) SaveResult(execID string, result *ExecutionResponse) error {
	resultPath := filepath.Join(em.execDir, execID+".result.json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(resultPath, data, 0644)
}

// LoadResult loads execution result from file
func (em *ExecutionManager) LoadResult(execID string) (*ExecutionResponse, error) {
	resultPath := filepath.Join(em.execDir, execID+".result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, err
	}

	var result ExecutionResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

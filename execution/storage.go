package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/not7/core/spec"
)

// Storage abstracts persistence operations for executions
type Storage interface {
	// Save persists an execution to storage
	Save(ctx context.Context, exec *Execution) error

	// Load retrieves an execution by ID
	Load(ctx context.Context, id string) (*Execution, error)

	// List returns all executions, sorted by creation time (newest first)
	List(ctx context.Context) ([]*ExecutionInfo, error)

	// SaveOutput writes the final output to a separate file for easy access
	SaveOutput(ctx context.Context, id string, output string) error

	// SaveTrace writes the full execution trace
	SaveTrace(ctx context.Context, id string, trace interface{}) error

	// Delete removes an execution from storage
	Delete(ctx context.Context, id string) error
}

// FileSystemStorage implements Storage using the local filesystem
type FileSystemStorage struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileSystemStorage creates a new filesystem-based storage
func NewFileSystemStorage(basePath string) (*FileSystemStorage, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileSystemStorage{
		basePath: basePath,
	}, nil
}

// Save persists an execution atomically to trace.json
func (s *FileSystemStorage) Save(ctx context.Context, exec *Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create execution directory
	execDir := s.executionDir(exec.ID)
	if err := os.MkdirAll(execDir, 0755); err != nil {
		return fmt.Errorf("failed to create execution directory: %w", err)
	}

	// Build trace data with execution metadata
	traceData := s.buildTraceData(exec)

	// Marshal to JSON
	data, err := json.MarshalIndent(traceData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution: %w", err)
	}

	// Write atomically: write to temp file, then rename
	traceFile := filepath.Join(execDir, "trace.json")
	tempFile := traceFile + ".tmp"

	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, traceFile); err != nil {
		os.Remove(tempFile) // Cleanup temp file on failure
		return fmt.Errorf("failed to commit trace file: %w", err)
	}

	return nil
}

// Load retrieves an execution by ID from trace.json
func (s *FileSystemStorage) Load(ctx context.Context, id string) (*Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	traceFile := filepath.Join(s.executionDir(id), "trace.json")

	data, err := os.ReadFile(traceFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrExecutionNotFound
		}
		return nil, fmt.Errorf("failed to read trace file: %w", err)
	}

	// Parse trace data structure
	var traceData map[string]interface{}
	if err := json.Unmarshal(data, &traceData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace: %w", err)
	}

	// Extract execution metadata
	exec, err := s.parseTraceData(traceData, id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trace data: %w", err)
	}

	return exec, nil
}

// List returns all executions sorted by creation time
func (s *FileSystemStorage) List(ctx context.Context) ([]*ExecutionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read executions directory: %w", err)
	}

	var infos []*ExecutionInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Load execution
		exec, err := s.loadWithoutLock(entry.Name())
		if err != nil {
			// Skip invalid executions
			continue
		}

		infos = append(infos, exec.Info())
	}

	// Sort by creation time (newest first)
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CreatedAt.After(infos[j].CreatedAt)
	})

	return infos, nil
}

// SaveOutput writes the final output to a text file
func (s *FileSystemStorage) SaveOutput(ctx context.Context, id string, output string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	execDir := s.executionDir(id)
	outputFile := filepath.Join(execDir, "output.txt")

	if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// SaveTrace is deprecated - trace.json is now written by Save()
// This method is kept for backward compatibility but does nothing
func (s *FileSystemStorage) SaveTrace(ctx context.Context, id string, trace interface{}) error {
	// No-op: trace.json is now written by Save()
	return nil
}

// Delete removes an execution and all its files
func (s *FileSystemStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	execDir := s.executionDir(id)

	if err := os.RemoveAll(execDir); err != nil {
		return fmt.Errorf("failed to delete execution: %w", err)
	}

	return nil
}

// executionDir returns the directory path for an execution
func (s *FileSystemStorage) executionDir(id string) string {
	return filepath.Join(s.basePath, id)
}

// loadWithoutLock loads an execution without acquiring the lock (internal use only)
func (s *FileSystemStorage) loadWithoutLock(id string) (*Execution, error) {
	traceFile := filepath.Join(s.executionDir(id), "trace.json")

	data, err := os.ReadFile(traceFile)
	if err != nil {
		return nil, err
	}

	var traceData map[string]interface{}
	if err := json.Unmarshal(data, &traceData); err != nil {
		return nil, err
	}

	exec, err := s.parseTraceData(traceData, id)
	if err != nil {
		return nil, err
	}

	return exec, nil
}

// buildTraceData constructs the enhanced trace structure with agent spec + execution metadata
func (s *FileSystemStorage) buildTraceData(exec *Execution) map[string]interface{} {
	trace := make(map[string]interface{})

	// Add all agent spec fields
	if exec.Spec != nil {
		specData, _ := json.Marshal(exec.Spec)
		json.Unmarshal(specData, &trace)
	}

	// Add execution metadata section
	metadata := map[string]interface{}{
		"execution_id": exec.ID,
		"status":       exec.Status,
		"created_at":   exec.CreatedAt,
	}

	if exec.StartedAt != nil {
		metadata["started_at"] = exec.StartedAt
	}
	if exec.EndedAt != nil {
		metadata["ended_at"] = exec.EndedAt
	}
	if exec.Result != nil {
		metadata["duration_ms"] = exec.Result.DurationMs
		metadata["total_cost"] = exec.Result.TotalCost
		if exec.Result.Error != "" {
			metadata["error"] = exec.Result.Error
		}
		// Include the full spec.Metadata from the result
		if exec.Result.Metadata != nil {
			metadata["executed_at"] = exec.Result.Metadata.ExecutedAt
			metadata["execution_time_ms"] = exec.Result.Metadata.ExecutionTimeMs
			metadata["node_results"] = exec.Result.Metadata.NodeResults
		}
	}

	trace["metadata"] = metadata

	return trace
}

// parseTraceData extracts execution info from trace structure
func (s *FileSystemStorage) parseTraceData(traceData map[string]interface{}, id string) (*Execution, error) {
	// Extract metadata section
	metadata, ok := traceData["metadata"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid metadata section")
	}

	// Parse status
	statusStr, _ := metadata["status"].(string)
	status := Status(statusStr)

	// Parse timestamps
	createdAt, _ := time.Parse(time.RFC3339, metadata["created_at"].(string))

	var startedAt, endedAt *time.Time
	if startedAtStr, ok := metadata["started_at"].(string); ok {
		t, _ := time.Parse(time.RFC3339, startedAtStr)
		startedAt = &t
	}
	if endedAtStr, ok := metadata["ended_at"].(string); ok {
		t, _ := time.Parse(time.RFC3339, endedAtStr)
		endedAt = &t
	}

	// Parse result if present
	var result *Result
	if durationMs, ok := metadata["duration_ms"].(float64); ok {
		result = &Result{
			DurationMs: int64(durationMs),
		}
		if totalCost, ok := metadata["total_cost"].(float64); ok {
			result.TotalCost = totalCost
		}
		if errorStr, ok := metadata["error"].(string); ok {
			result.Error = errorStr
		}
	}

	// Parse agent spec (all fields except metadata)
	specData := make(map[string]interface{})
	for k, v := range traceData {
		if k != "metadata" {
			specData[k] = v
		}
	}
	specBytes, _ := json.Marshal(specData)
	var agentSpec spec.AgentSpec
	json.Unmarshal(specBytes, &agentSpec)

	// Construct execution
	exec := &Execution{
		ID:        id,
		Spec:      &agentSpec,
		Status:    status,
		Result:    result,
		CreatedAt: createdAt,
		StartedAt: startedAt,
		EndedAt:   endedAt,
	}

	return exec, nil
}

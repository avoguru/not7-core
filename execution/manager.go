package execution

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/not7/core/executor"
	"github.com/not7/core/logger"
	"github.com/not7/core/spec"
)

// Manager orchestrates agent executions with thread-safe operations
type Manager struct {
	storage Storage
	logDir  string

	// Track active executions for concurrent safety
	activeExecutions sync.Map // map[string]*Execution

	// Protect state mutations
	mu sync.RWMutex
}

// NewManager creates a new execution manager
func NewManager(storage Storage, logDir string) *Manager {
	return &Manager{
		storage: storage,
		logDir:  logDir,
	}
}

// Execute runs an agent with the specified options
// For async execution, it returns immediately with execution ID
// For sync execution, it blocks until completion
func (m *Manager) Execute(ctx context.Context, agentSpec *spec.AgentSpec, opts Options) (*Execution, error) {
	// Validate spec
	if err := spec.ValidateSpec(agentSpec); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSpec, err)
	}

	// Generate unique execution ID
	execID := m.generateExecutionID(agentSpec)

	// Create execution instance
	exec := NewExecution(execID, agentSpec)

	// Save initial state
	if err := m.storage.Save(ctx, exec); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}

	// Track as active
	if _, loaded := m.activeExecutions.LoadOrStore(execID, exec); loaded {
		return nil, ErrExecutionAlreadyRunning
	}

	if opts.Async {
		// Execute asynchronously
		go m.executeAsync(context.Background(), exec, opts)
		return exec, nil
	}

	// Execute synchronously
	return m.executeSync(ctx, exec, opts)
}

// executeSync performs synchronous execution with optional trace streaming
func (m *Manager) executeSync(ctx context.Context, exec *Execution, opts Options) (*Execution, error) {
	defer m.activeExecutions.Delete(exec.ID)

	// Mark as started
	exec.MarkStarted()
	if err := m.storage.Save(ctx, exec); err != nil {
		return nil, err
	}

	// Create logger for this execution
	log, err := logger.NewFileLogger(m.logDir, exec.ID)
	if err != nil {
		exec.MarkFailed(fmt.Errorf("failed to create logger: %w", err))
		m.storage.Save(ctx, exec)
		return exec, err
	}
	defer log.Close()

	log.Info("Starting execution: %s", exec.Spec.Goal)
	log.Info("Execution ID: %s", exec.ID)

	// Create and configure executor
	execEngine, err := executor.NewExecutorWithLogger(exec.Spec, log)
	if err != nil {
		exec.MarkFailed(fmt.Errorf("failed to create executor: %w", err))
		m.storage.Save(ctx, exec)
		return exec, err
	}

	// Execute with timeout if specified
	execCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Execute agent
	startTime := time.Now()
	output, execErr := m.runWithContext(execCtx, execEngine)
	duration := time.Since(startTime)

	// Build result
	result := &Result{
		Output:     output,
		DurationMs: duration.Milliseconds(),
	}

	if execErr != nil {
		result.Error = execErr.Error()
		exec.MarkFailed(execErr)
		log.Error("Execution failed: %v", execErr)
	} else {
		// Get metadata from executor
		metadata := execEngine.GetMetadata()
		result.Metadata = metadata
		result.TotalCost = metadata.TotalCost

		exec.MarkCompleted(result)
		log.Info("Execution completed: duration=%dms, cost=$%.4f", result.DurationMs, result.TotalCost)
	}

	// Save final state
	if err := m.storage.Save(ctx, exec); err != nil {
		log.Error("Failed to save execution result: %v", err)
	}

	// Save output separately for easy access
	if output != "" {
		if err := m.storage.SaveOutput(ctx, exec.ID, output); err != nil {
			log.Error("Failed to save output file: %v", err)
		}
	}

	// Save trace with full metadata
	if err := m.storage.SaveTrace(ctx, exec.ID, exec.Spec); err != nil {
		log.Error("Failed to save trace: %v", err)
	}

	return exec, execErr
}

// executeAsync performs asynchronous execution in a goroutine
func (m *Manager) executeAsync(ctx context.Context, exec *Execution, opts Options) {
	defer m.activeExecutions.Delete(exec.ID)

	// Execute synchronously within the goroutine
	// We use a background context since the caller has already returned
	m.executeSync(ctx, exec, opts)
}

// runWithContext executes the agent with context support
func (m *Manager) runWithContext(ctx context.Context, exec *executor.Executor) (string, error) {
	// Create a channel to receive the result
	type execResult struct {
		output string
		err    error
	}
	resultCh := make(chan execResult, 1)

	// Run executor in goroutine
	go func() {
		output, err := exec.Execute("")
		resultCh <- execResult{output: output, err: err}
	}()

	// Wait for either completion or cancellation
	select {
	case <-ctx.Done():
		return "", ErrExecutionCancelled
	case result := <-resultCh:
		return result.output, result.err
	}
}

// GetExecution retrieves an execution by ID
func (m *Manager) GetExecution(ctx context.Context, id string) (*Execution, error) {
	// Check if it's active
	if exec, ok := m.activeExecutions.Load(id); ok {
		return exec.(*Execution), nil
	}

	// Load from storage
	return m.storage.Load(ctx, id)
}

// ListExecutions returns all executions
func (m *Manager) ListExecutions(ctx context.Context) ([]*ExecutionInfo, error) {
	return m.storage.List(ctx)
}

// DeleteExecution removes an execution
func (m *Manager) DeleteExecution(ctx context.Context, id string) error {
	// Check if running
	if _, ok := m.activeExecutions.Load(id); ok {
		return fmt.Errorf("cannot delete running execution")
	}

	return m.storage.Delete(ctx, id)
}

// GetStatus returns the current status of an execution
func (m *Manager) GetStatus(ctx context.Context, id string) (Status, error) {
	exec, err := m.GetExecution(ctx, id)
	if err != nil {
		return "", err
	}
	return exec.Status, nil
}

// generateExecutionID creates a unique execution ID
func (m *Manager) generateExecutionID(agentSpec *spec.AgentSpec) string {
	timestamp := time.Now().UnixNano()

	if agentSpec.ID != "" {
		return fmt.Sprintf("%s-%d", agentSpec.ID, timestamp)
	}

	return fmt.Sprintf("exec-%d", timestamp)
}

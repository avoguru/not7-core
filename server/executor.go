package server

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/not7/core/executor"
	"github.com/not7/core/logger"
	"github.com/not7/core/spec"
)

// executeAgentWithID executes an agent with logging and returns execution response
func (s *Server) executeAgentWithID(executionID string, agentSpec *spec.AgentSpec) (*ExecutionResponse, error) {
	// Create logger
	log, err := logger.NewFileLogger(s.logDir, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	defer log.Close()

	log.Info("Executing agent: %s", agentSpec.Goal)
	if agentSpec.ID != "" {
		log.Info("Agent ID: %s", agentSpec.ID)
	}

	// Create executor
	exec, err := executor.NewExecutorWithLogger(agentSpec, log)
	if err != nil {
		log.Error("Failed to create executor: %v", err)
		return nil, err
	}

	// Execute
	startTime := time.Now()
	output, err := exec.Execute("")
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Error("Execution failed: %v", err)
		return nil, err
	}

	metadata := exec.GetMetadata()
	log.Info("Execution completed in %dms, cost: $%.4f", duration, metadata.TotalCost)

	return &ExecutionResponse{
		ID:       executionID,
		Status:   "success",
		Goal:     agentSpec.Goal,
		Output:   output,
		Cost:     metadata.TotalCost,
		Duration: duration,
		LogFile:  log.LogFilePath(),
	}, nil
}

// executeAgentFromFile executes an agent from a file path (used by file watcher)
func (s *Server) executeAgentFromFile(filePath string) {
	executionID := fmt.Sprintf("file-%d", time.Now().UnixNano())

	// Create logger
	log, err := logger.NewFileLogger(s.logDir, executionID)
	if err != nil {
		fmt.Printf("[File Watcher] Failed to create logger: %v\n", err)
		return
	}
	defer log.Close()

	log.Info("Executing agent from file: %s", filePath)

	// Load spec
	agentSpec, err := spec.LoadSpec(filePath)
	if err != nil {
		log.Error("Failed to load spec: %v", err)
		fmt.Printf("[File Watcher] Failed to load %s: %v\n", filepath.Base(filePath), err)
		return
	}

	// Validate
	if err := spec.ValidateSpec(agentSpec); err != nil {
		log.Error("Validation failed: %v", err)
		fmt.Printf("[File Watcher] Validation failed for %s: %v\n", filepath.Base(filePath), err)
		return
	}

	log.Info("Goal: %s", agentSpec.Goal)

	// Execute
	exec, err := executor.NewExecutorWithLogger(agentSpec, log)
	if err != nil {
		log.Error("Failed to create executor: %v", err)
		return
	}

	startTime := time.Now()
	output, err := exec.Execute("")
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Error("Execution failed: %v", err)
		fmt.Printf("[File Watcher] Execution failed for %s: %v\n", filepath.Base(filePath), err)
		return
	}

	log.Info("Execution completed in %dms", duration)
	log.Info("Output length: %d characters", len(output))
	metadata := exec.GetMetadata()
	log.Info("Total cost: $%.4f", metadata.TotalCost)

	fmt.Printf("[File Watcher] ✅ Completed %s in %dms (cost: $%.4f)\n",
		filepath.Base(filePath), duration, metadata.TotalCost)
}

// executeAsyncAgent runs an agent in the background with status tracking
func (s *Server) executeAsyncAgent(executionID string, agentSpec *spec.AgentSpec) {
	startTime := time.Now()
	
	// Update status to running
	s.execMgr.UpdateStatus(executionID, "running")
	
	fmt.Printf("[Async] Started execution: %s\n", executionID)
	
	// Execute the agent
	result, err := s.executeAgentWithID(executionID, agentSpec)
	
	if err != nil {
		// Mark as failed
		s.execMgr.UpdateStatus(executionID, "failed")
		
		// Save error result
		errorResult := &ExecutionResponse{
			ID:     executionID,
			Status: "failed",
			Error:  err.Error(),
		}
		s.execMgr.SaveResult(executionID, errorResult)
		
		fmt.Printf("[Async] Failed execution: %s - %v\n", executionID, err)
		return
	}
	
	// Mark as completed
	s.execMgr.UpdateStatus(executionID, "completed")
	
	// Update final metrics
	elapsed := time.Since(startTime).Milliseconds()
	s.execMgr.UpdateProgress(executionID, len(agentSpec.Nodes), "", "", result.Cost, elapsed)
	
	// Save result
	if err := s.execMgr.SaveResult(executionID, result); err != nil {
		fmt.Printf("[Async] Failed to save result: %v\n", err)
	}
	
	fmt.Printf("[Async] Completed execution: %s (%.2fs, $%.4f)\n", 
		executionID, float64(elapsed)/1000, result.Cost)
}

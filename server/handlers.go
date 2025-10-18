package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/not7/core/execution"
	"github.com/not7/core/spec"
)

// handleRun handles POST /api/v1/run - Execute agent
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "", "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse agent spec
	var agentSpec spec.AgentSpec
	if err := json.Unmarshal(body, &agentSpec); err != nil {
		respondError(w, "", "Invalid JSON specification", http.StatusBadRequest)
		return
	}

	// Parse options from query parameters
	opts := execution.Options{
		Async:  r.URL.Query().Get("async") == "true",
		Stream: r.URL.Query().Get("stream") == "true",
	}

	fmt.Printf("[API] Executing agent: %s (async=%v, stream=%v)\n", agentSpec.Goal, opts.Async, opts.Stream)

	// Execute through manager
	ctx := context.Background()
	exec, err := s.execMgr.Execute(ctx, &agentSpec, opts)

	if err != nil {
		respondError(w, "", fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// For async, return immediately with execution ID
	if opts.Async {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution_id": exec.ID,
			"status":       exec.Status,
			"message":      "Execution started in background",
		})
		return
	}

	// For sync, return full result
	response := buildExecutionResponse(exec)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleExecutions handles execution-related requests
func (s *Server) handleExecutions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/executions/")

	// GET /executions - list all
	if path == "" || path == "/" {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.listExecutions(w, r)
		return
	}

	// GET /executions/{id} - get specific execution
	execID := strings.TrimSuffix(path, "/")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.getExecution(w, r, execID)
}

// listExecutions handles GET /api/v1/executions
func (s *Server) listExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	executions, err := s.execMgr.ListExecutions(ctx)
	if err != nil {
		respondError(w, "", fmt.Sprintf("Failed to list executions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	})
}

// getExecution handles GET /api/v1/executions/{id}
func (s *Server) getExecution(w http.ResponseWriter, r *http.Request, execID string) {
	ctx := context.Background()
	exec, err := s.execMgr.GetExecution(ctx, execID)

	if err != nil {
		if err == execution.ErrExecutionNotFound {
			respondError(w, execID, "Execution not found", http.StatusNotFound)
		} else {
			respondError(w, execID, fmt.Sprintf("Failed to get execution: %v", err), http.StatusInternalServerError)
		}
		return
	}

	response := buildExecutionResponse(exec)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"server": "NOT7",
	})
}

// buildExecutionResponse converts execution domain model to API response
func buildExecutionResponse(exec *execution.Execution) map[string]interface{} {
	response := map[string]interface{}{
		"id":         exec.ID,
		"status":     exec.Status,
		"goal":       exec.Spec.Goal,
		"created_at": exec.CreatedAt,
	}

	if exec.StartedAt != nil {
		response["started_at"] = exec.StartedAt
	}

	if exec.EndedAt != nil {
		response["ended_at"] = exec.EndedAt
	}

	if exec.Result != nil {
		response["output"] = exec.Result.Output
		response["duration_ms"] = exec.Result.DurationMs
		response["total_cost"] = exec.Result.TotalCost

		if exec.Result.Error != "" {
			response["error"] = exec.Result.Error
		}

		if exec.Result.Metadata != nil {
			response["metadata"] = exec.Result.Metadata
		}
	}

	return response
}

// respondError sends a standardized error response
func respondError(w http.ResponseWriter, id, message string, statusCode int) {
	response := map[string]interface{}{
		"status": "error",
		"error":  message,
	}
	if id != "" {
		response["id"] = id
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

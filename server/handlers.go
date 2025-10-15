package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/not7/core/spec"
)

// handleAgents handles GET (list) and POST (create) for /api/v1/agents
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listAgents(w, r)
	case http.MethodPost:
		s.createAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAgentOperations handles GET/PUT/DELETE/POST(run) for /api/v1/agents/{id}
func (s *Server) handleAgentOperations(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")

	// Handle /agents/{id}/run
	if strings.HasSuffix(path, "/run") {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimSuffix(path, "/run")
		s.runAgentByID(w, r, id)
		return
	}

	// Handle /agents/{id}
	id := path
	switch r.Method {
	case http.MethodGet:
		s.getAgent(w, r, id)
	case http.MethodPut:
		s.updateAgent(w, r, id)
	case http.MethodDelete:
		s.deleteAgent(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createAgent handles POST /api/v1/agents - Save agent without executing
func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, "", "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var agentSpec spec.AgentSpec
	if err := json.Unmarshal(body, &agentSpec); err != nil {
		respondError(w, "", "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate ID is provided
	if agentSpec.ID == "" {
		respondError(w, "", "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Validate spec
	if err := spec.ValidateSpec(&agentSpec); err != nil {
		respondError(w, agentSpec.ID, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Check if agent already exists
	if s.agentExists(agentSpec.ID) {
		respondError(w, agentSpec.ID, "Agent already exists. Use PUT to update.", http.StatusConflict)
		return
	}

	// Save agent spec
	if err := s.saveAgentSpec(agentSpec.ID, &agentSpec); err != nil {
		respondError(w, agentSpec.ID, fmt.Sprintf("Failed to save agent: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[API] Agent deployed: %s\n", agentSpec.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     agentSpec.ID,
		"status": "deployed",
		"goal":   agentSpec.Goal,
	})
}

// listAgents handles GET /api/v1/agents - List all deployed agents
func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.listAgentSpecs()
	if err != nil {
		respondError(w, "", fmt.Sprintf("Failed to list agents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
		"count":  len(agents),
	})
}

// getAgent handles GET /api/v1/agents/{id} - Get specific agent
func (s *Server) getAgent(w http.ResponseWriter, r *http.Request, id string) {
	agentSpec, err := s.loadAgentSpec(id)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, id, "Agent not found", http.StatusNotFound)
		} else {
			respondError(w, id, fmt.Sprintf("Failed to load agent: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentSpec)
}

// updateAgent handles PUT /api/v1/agents/{id} - Update agent
func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request, id string) {
	// Check if agent exists
	if !s.agentExists(id) {
		respondError(w, id, "Agent not found. Use POST to create.", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, id, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var agentSpec spec.AgentSpec
	if err := json.Unmarshal(body, &agentSpec); err != nil {
		respondError(w, id, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Ensure ID matches
	agentSpec.ID = id

	// Validate spec
	if err := spec.ValidateSpec(&agentSpec); err != nil {
		respondError(w, id, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Save updated spec
	if err := s.saveAgentSpec(id, &agentSpec); err != nil {
		respondError(w, id, fmt.Sprintf("Failed to update agent: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[API] Agent updated: %s\n", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"status": "updated",
		"goal":   agentSpec.Goal,
	})
}

// deleteAgent handles DELETE /api/v1/agents/{id} - Delete agent
func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request, id string) {
	if !s.agentExists(id) {
		respondError(w, id, "Agent not found", http.StatusNotFound)
		return
	}

	if err := s.deleteAgentSpec(id); err != nil {
		respondError(w, id, fmt.Sprintf("Failed to delete agent: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[API] Agent deleted: %s\n", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"status": "deleted",
	})
}

// runAgentByID handles POST /api/v1/agents/{id}/run - Execute deployed agent
func (s *Server) runAgentByID(w http.ResponseWriter, r *http.Request, id string) {
	agentSpec, err := s.loadAgentSpec(id)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, id, "Agent not found", http.StatusNotFound)
		} else {
			respondError(w, id, fmt.Sprintf("Failed to load agent: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Execute the agent
	executionID := fmt.Sprintf("%s-%d", id, time.Now().UnixNano())

	// Check for async execution
	asyncMode := r.URL.Query().Get("async") == "true"

	if asyncMode {
		// Create execution status
		status := s.execMgr.CreateExecution(executionID, agentSpec)

		// Start background execution
		go s.executeAsyncAgent(executionID, agentSpec)

		// Return 202 Accepted with execution ID
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution_id": executionID,
			"status":       status.Status,
			"message":      "Execution started in background",
		})
		return
	}

	// Synchronous execution (original behavior)
	result, err := s.executeAgentWithID(executionID, agentSpec)

	if err != nil {
		respondError(w, executionID, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleRunAnonymous handles POST /api/v1/agents/run - Execute anonymous agent
func (s *Server) handleRunAnonymous(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	executionID := fmt.Sprintf("anon-%d", time.Now().UnixNano())

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, executionID, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var agentSpec spec.AgentSpec
	if err := json.Unmarshal(body, &agentSpec); err != nil {
		respondError(w, executionID, "Invalid JSON specification", http.StatusBadRequest)
		return
	}

	// Validate spec
	if err := spec.ValidateSpec(&agentSpec); err != nil {
		respondError(w, executionID, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Check for async execution
	asyncMode := r.URL.Query().Get("async") == "true"

	if asyncMode {
		// Create execution status
		status := s.execMgr.CreateExecution(executionID, &agentSpec)

		// Start background execution
		go s.executeAsyncAgent(executionID, &agentSpec)

		// Return 202 Accepted with execution ID
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution_id": executionID,
			"status":       status.Status,
			"message":      "Execution started in background",
		})
		return
	}

	// Synchronous execution (original behavior)
	result, err := s.executeAgentWithID(executionID, &agentSpec)
	if err != nil {
		respondError(w, executionID, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"server": "NOT7",
	})
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

	// Extract execution ID and action
	parts := strings.Split(path, "/")
	execID := parts[0]

	if len(parts) == 1 {
		// GET /executions/{id} - assume they want status
		if r.Method == http.MethodGet {
			s.getExecutionStatus(w, r, execID)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := parts[1]
	switch action {
	case "status":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.getExecutionStatus(w, r, execID)
	case "result":
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.getExecutionResult(w, r, execID)
	default:
		http.Error(w, "Unknown action", http.StatusNotFound)
	}
}

// listExecutions handles GET /api/v1/executions
func (s *Server) listExecutions(w http.ResponseWriter, r *http.Request) {
	executions := s.execMgr.ListExecutions()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"executions": executions,
		"count":      len(executions),
	})
}

// getExecutionStatus handles GET /api/v1/executions/{id}/status
func (s *Server) getExecutionStatus(w http.ResponseWriter, r *http.Request, execID string) {
	status, err := s.execMgr.GetStatus(execID)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, execID, "Execution not found", http.StatusNotFound)
		} else {
			respondError(w, execID, fmt.Sprintf("Failed to get status: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getExecutionResult handles GET /api/v1/executions/{id}/result
func (s *Server) getExecutionResult(w http.ResponseWriter, r *http.Request, execID string) {
	result, err := s.execMgr.LoadResult(execID)
	if err != nil {
		if os.IsNotExist(err) {
			respondError(w, execID, "Result not available yet or execution not found", http.StatusNotFound)
		} else {
			respondError(w, execID, fmt.Sprintf("Failed to load result: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

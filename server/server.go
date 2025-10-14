package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/not7/core/config"
	"github.com/not7/core/executor"
	"github.com/not7/core/logger"
	"github.com/not7/core/spec"
)

// Server represents the NOT7 HTTP server
type Server struct {
	port      int
	deployDir string
	logDir    string
}

// NewServer creates a new NOT7 server
func NewServer(port int, deployDir, logDir string) *Server {
	if port == 0 {
		port = 8080
	}
	if deployDir == "" {
		deployDir = "./deploy"
	}
	if logDir == "" {
		logDir = "./logs"
	}
	return &Server{
		port:      port,
		deployDir: deployDir,
		logDir:    logDir,
	}
}

// ExecutionResponse represents the API response
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

// Start starts the HTTP server and file watcher
func (s *Server) Start() error {
	// Create necessary directories
	if err := os.MkdirAll(s.deployDir, 0755); err != nil {
		return fmt.Errorf("failed to create deploy directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(s.deployDir, "processed"), 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}
	if err := os.MkdirAll(s.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Set up HTTP handlers
	http.HandleFunc("/api/v1/agents/run", s.handleRunAgent)
	http.HandleFunc("/health", s.handleHealth)

	// Start file watcher in background
	go s.watchDeployFolder()

	// Display startup info
	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("NOT7 - Agent Runtime Server\n")
	fmt.Printf("===========================\n\n")
	fmt.Printf("🚀 Server listening on http://localhost%s\n", addr)
	fmt.Printf("📁 Deploy directory: %s\n", s.deployDir)
	fmt.Printf("📁 Logs directory: %s\n", s.logDir)
	fmt.Printf("📖 API endpoint: POST /api/v1/agents/run\n")
	fmt.Printf("👀 Watching %s for new agent specs...\n\n", s.deployDir)

	// Start HTTP server
	return http.ListenAndServe(addr, nil)
}

// watchDeployFolder watches for new JSON files in the deploy directory
func (s *Server) watchDeployFolder() {
	cfg := config.Get()
	pollInterval := time.Duration(cfg.Watcher.PollIntervalSeconds) * time.Second

	fmt.Printf("[File Watcher] Started watching: %s\n", s.deployDir)
	fmt.Printf("[File Watcher] Poll interval: %v\n", pollInterval)

	for {
		files, err := os.ReadDir(s.deployDir)
		if err != nil {
			fmt.Printf("[File Watcher] Error reading deploy directory: %v\n", err)
			time.Sleep(pollInterval)
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			filePath := filepath.Join(s.deployDir, file.Name())
			fmt.Printf("[File Watcher] Detected new agent: %s\n", file.Name())

			// Execute the agent
			s.executeAgentFromFile(filePath)

			// Move to processed folder
			processedPath := filepath.Join(s.deployDir, "processed", file.Name())
			if err := os.Rename(filePath, processedPath); err != nil {
				fmt.Printf("[File Watcher] Warning: Failed to move file to processed: %v\n", err)
			} else {
				fmt.Printf("[File Watcher] Moved to processed: %s\n", file.Name())
			}
		}

		time.Sleep(pollInterval)
	}
}

// executeAgentFromFile executes an agent from a file path
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

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"server": "NOT7",
	})
}

// handleRunAgent handles agent execution requests
func (s *Server) handleRunAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate execution ID
	executionID := fmt.Sprintf("api-%d", time.Now().UnixNano())

	// Create logger for this execution
	log, err := logger.NewFileLogger(s.logDir, executionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create logger: %v", err), http.StatusInternalServerError)
		return
	}
	defer log.Close()

	log.Info("Received agent execution request via API")
	log.Info("Execution ID: %s", executionID)

	// Read and parse spec from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body: %v", err)
		respondError(w, executionID, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var agentSpec spec.AgentSpec
	if err := json.Unmarshal(body, &agentSpec); err != nil {
		log.Error("Failed to parse agent spec: %v", err)
		respondError(w, executionID, "Invalid JSON specification", http.StatusBadRequest)
		return
	}

	// Save to deploy directory (for record keeping)
	timestamp := time.Now().Format("20060102-150405")
	deployFile := filepath.Join(s.deployDir, "processed", fmt.Sprintf("api-%s-%s.json", timestamp, executionID))
	if err := spec.SaveSpec(&agentSpec, deployFile); err != nil {
		log.Error("Warning: Failed to save spec to deploy folder: %v", err)
	} else {
		log.Info("Saved spec to: %s", deployFile)
	}

	// Validate spec
	if err := spec.ValidateSpec(&agentSpec); err != nil {
		log.Error("Spec validation failed: %v", err)
		respondError(w, executionID, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	log.Info("Agent spec validated successfully")
	log.Info("Goal: %s", agentSpec.Goal)
	log.Info("Nodes: %d", len(agentSpec.Nodes))

	// Create executor
	exec, err := executor.NewExecutorWithLogger(&agentSpec, log)
	if err != nil {
		log.Error("Failed to create executor: %v", err)
		respondError(w, executionID, fmt.Sprintf("Executor creation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute agent
	log.Info("Starting agent execution")
	startTime := time.Now()

	output, err := exec.Execute("")
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Error("Execution failed: %v", err)
		respondError(w, executionID, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Info("Execution completed successfully")
	log.Info("Duration: %dms", duration)

	metadata := exec.GetMetadata()
	log.Info("Total cost: $%.4f", metadata.TotalCost)

	// Send success response
	response := ExecutionResponse{
		ID:       executionID,
		Status:   "success",
		Goal:     agentSpec.Goal,
		Output:   output,
		Cost:     metadata.TotalCost,
		Duration: duration,
		LogFile:  log.LogFilePath(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	log.Info("Response sent to client")
}

// respondError sends an error response
func respondError(w http.ResponseWriter, id, message string, statusCode int) {
	response := ExecutionResponse{
		ID:     id,
		Status: "error",
		Error:  message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// Server represents the NOT7 HTTP server
type Server struct {
	port      int
	deployDir string
	logDir    string
	execMgr   *ExecutionManager
}

// NewServer creates a new NOT7 server instance
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

	execDir := filepath.Join(deployDir, "executions")

	return &Server{
		port:      port,
		deployDir: deployDir,
		logDir:    logDir,
		execMgr:   NewExecutionManager(execDir),
	}
}

// Start initializes directories, registers HTTP handlers, and starts the server
func (s *Server) Start() error {
	// Create necessary directories
	if err := os.MkdirAll(filepath.Join(s.deployDir, "specs"), 0755); err != nil {
		return fmt.Errorf("failed to create specs directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(s.deployDir, "processed"), 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}
	if err := os.MkdirAll(s.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	if err := os.MkdirAll(s.execMgr.execDir, 0755); err != nil {
		return fmt.Errorf("failed to create executions directory: %w", err)
	}

	// Register HTTP handlers
	http.HandleFunc("/api/v1/agents/run", s.handleRunAnonymous) // Anonymous execution
	http.HandleFunc("/api/v1/agents/", s.handleAgentOperations) // Must be after /run
	http.HandleFunc("/api/v1/agents", s.handleAgents)
	http.HandleFunc("/api/v1/executions/", s.handleExecutions) // Execution status/results
	http.HandleFunc("/health", s.handleHealth)

	// Start file watcher in background
	go s.watchDeployFolder()

	// Display startup information
	s.printStartupInfo()

	// Start HTTP server (blocks until error)
	addr := fmt.Sprintf(":%d", s.port)
	return http.ListenAndServe(addr, nil)
}

// printStartupInfo displays server configuration and available endpoints
func (s *Server) printStartupInfo() {
	fmt.Println()
	fmt.Println("╔═════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                             ║")
	fmt.Println("║            ███╗   ██╗ ██████╗ ████████╗███████╗             ║")
	fmt.Println("║            ████╗  ██║██╔═══██╗╚══██╔══╝╚════██║             ║")
	fmt.Println("║            ██╔██╗ ██║██║   ██║   ██║       ██╔╝             ║")
	fmt.Println("║            ██║╚██╗██║██║   ██║   ██║      ██╔╝              ║")
	fmt.Println("║            ██║ ╚████║╚██████╔╝   ██║      ██║               ║")
	fmt.Println("║            ╚═╝  ╚═══╝ ╚═════╝    ╚═╝      ╚═╝               ║")
	fmt.Println("║                                                             ║")
	fmt.Println("║                 Declarative Agent Runtime                   ║")
	fmt.Println("║                     https://not7.ai                         ║")
	fmt.Println("║                                                             ║")
	fmt.Println("╚═════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("🚀 Server listening on http://localhost:%d\n", s.port)
	fmt.Printf("📁 Specs directory: %s/specs\n", s.deployDir)
	fmt.Printf("📁 Deploy directory: %s\n", s.deployDir)
	fmt.Printf("📁 Logs directory: %s\n", s.logDir)
	fmt.Printf("\n📖 API Endpoints:\n")
	fmt.Printf("   POST   /api/v1/agents          - Deploy agent\n")
	fmt.Printf("   GET    /api/v1/agents          - List agents\n")
	fmt.Printf("   GET    /api/v1/agents/{id}     - Get agent\n")
	fmt.Printf("   PUT    /api/v1/agents/{id}     - Update agent\n")
	fmt.Printf("   DELETE /api/v1/agents/{id}     - Delete agent\n")
	fmt.Printf("   POST   /api/v1/agents/{id}/run - Execute agent\n")
	fmt.Printf("   POST   /api/v1/agents/run      - Execute anonymous agent\n")
	fmt.Printf("\n👀 Watching %s for new agent specs...\n\n", s.deployDir)
}

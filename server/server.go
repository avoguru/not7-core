package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/not7/core/execution"
)

// Server represents the NOT7 HTTP server
type Server struct {
	port       int
	execMgr    *execution.Manager
	logDir     string
	execDir    string
}

// NewServer creates a new NOT7 server instance
func NewServer(port int, execDir, logDir string) *Server {
	if port == 0 {
		port = 8080
	}
	if execDir == "" {
		execDir = "./executions"
	}
	if logDir == "" {
		logDir = "./logs"
	}

	// Create storage
	storage, err := execution.NewFileSystemStorage(execDir)
	if err != nil {
		panic(fmt.Errorf("failed to create storage: %w", err))
	}

	return &Server{
		port:    port,
		execMgr: execution.NewManager(storage, logDir),
		logDir:  logDir,
		execDir: execDir,
	}
}

// Start initializes directories, registers HTTP handlers, and starts the server
func (s *Server) Start() error {
	// Create necessary directories
	if err := os.MkdirAll(s.execDir, 0755); err != nil {
		return fmt.Errorf("failed to create executions directory: %w", err)
	}
	if err := os.MkdirAll(s.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Register HTTP handlers
	http.HandleFunc("/api/v1/run", s.handleRun)             // Primary execution endpoint
	http.HandleFunc("/api/v1/executions/", s.handleExecutions) // Execution status/results
	http.HandleFunc("/health", s.handleHealth)

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
	fmt.Printf("📁 Executions: %s\n", s.execDir)
	fmt.Printf("📁 Logs: %s\n", s.logDir)
	fmt.Printf("\n📖 API Endpoints:\n")
	fmt.Printf("   POST   /api/v1/run                  - Execute agent\n")
	fmt.Printf("   GET    /api/v1/executions           - List executions\n")
	fmt.Printf("   GET    /api/v1/executions/{id}      - Get execution status\n")
	fmt.Printf("   GET    /health                      - Health check\n")
	fmt.Printf("\n💡 Usage:\n")
	fmt.Printf("   CLI:  ./not7 run agent.json\n")
	fmt.Printf("   API:  curl -X POST http://localhost:%d/api/v1/run -d @agent.json\n", s.port)
	fmt.Println()
}

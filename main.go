package main

import (
	"fmt"
	"os"

	"github.com/not7/core/client"
	"github.com/not7/core/config"
	"github.com/not7/core/server"
	"github.com/not7/core/spec"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Only server needs config loaded upfront
	if command == "serve" {
		loadConfig()
	}

	switch command {
	case "serve":
		runServer()
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Error: spec file required")
			printUsage()
			os.Exit(1)
		}
		runAgent(os.Args[2])
	case "status":
		if len(os.Args) < 3 {
			fmt.Println("Error: execution ID required")
			printUsage()
			os.Exit(1)
		}
		getStatus(os.Args[2])
	case "result":
		if len(os.Args) < 3 {
			fmt.Println("Error: execution ID required")
			printUsage()
			os.Exit(1)
		}
		getResult(os.Args[2])
	case "agents":
		listAgents()
	case "validate":
		if len(os.Args) < 3 {
			fmt.Println("Error: spec file required")
			printUsage()
			os.Exit(1)
		}
		validateSpec(os.Args[2])
	case "--help", "-h", "help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func loadConfig() {
	configFile := "not7.conf"
	if envConfig := os.Getenv("NOT7_CONFIG"); envConfig != "" {
		configFile = envConfig
	}
	
	if _, err := config.LoadConfig(configFile); err != nil {
		fmt.Printf("❌ Failed to load config from %s: %v\n", configFile, err)
		fmt.Println("\nPlease copy not7.conf.example to not7.conf and update with your API key:")
		fmt.Println("  cp not7.conf.example not7.conf")
		fmt.Println("  # Then edit not7.conf with your OpenAI API key")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`
NOT7 - Agent Runtime

Usage:
  not7 serve                  Start server (required)
  not7 run <agent.json>       Execute agent
  not7 status <exec-id>       Check execution status
  not7 result <exec-id>       Get execution result
  not7 agents                 List deployed agents
  not7 validate <agent.json>  Validate spec (offline)

Workflow:
  Terminal 1: not7 serve                    # Start server
  Terminal 2: not7 run agent.json           # Execute
  Terminal 2: not7 status exec-123          # Check progress
  Terminal 2: not7 result exec-123          # Get result

Examples:
  # Terminal 1
  not7 serve

  # Terminal 2
  not7 run examples/poem-generator.json
  not7 run examples/problem-solver.json --async
  not7 status exec-1234567890
  not7 result exec-1234567890
  not7 agents
`)
}

func runServer() {
	cfg := config.Get()
	srv := server.NewServer(cfg.Server.Port, cfg.Server.DeployDir, cfg.Server.LogDir)
	if err := srv.Start(); err != nil {
		fmt.Printf("❌ Server error: %v\n", err)
		os.Exit(1)
	}
}

func runAgent(specFile string) {
	apiClient := client.NewClient("")
	
	if err := apiClient.CheckHealth(); err != nil {
		fmt.Println("❌ Server not running. Start server first:")
		fmt.Println("  Terminal 1: not7 serve")
		fmt.Println("  Terminal 2: not7 run agent.json")
		os.Exit(1)
	}
	
	agentJSON, err := os.ReadFile(specFile)
	if err != nil {
		fmt.Printf("❌ Failed to read spec: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("📖 Executing: %s\n", specFile)
	
	asyncMode := len(os.Args) > 3 && os.Args[3] == "--async"
	
	result, err := apiClient.RunAgent(agentJSON, asyncMode)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	
	if asyncMode {
		fmt.Printf("\n✅ Submitted (background)\n")
		fmt.Printf("📋 Execution ID: %s\n\n", result["execution_id"])
		fmt.Printf("Check: not7 status %s\n", result["execution_id"])
	} else {
		printExecutionResult(result)
	}
}

func getStatus(execID string) {
	apiClient := client.NewClient("")
	
	if err := apiClient.CheckHealth(); err != nil {
		fmt.Println("❌ Server not running")
		os.Exit(1)
	}
	
	status, err := apiClient.GetExecutionStatus(execID)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Execution: %s\n", execID)
	fmt.Printf("Status: %s\n", status["status"])
	fmt.Printf("Goal: %s\n", status["goal"])
	
	if progress, ok := status["progress"].(map[string]interface{}); ok {
		fmt.Printf("Progress: %v/%v nodes\n", 
			progress["completed_nodes"], progress["total_nodes"])
	}
}

func getResult(execID string) {
	apiClient := client.NewClient("")
	
	if err := apiClient.CheckHealth(); err != nil {
		fmt.Println("❌ Server not running")
		os.Exit(1)
	}
	
	result, err := apiClient.GetExecutionResult(execID)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	
	printExecutionResult(result)
}

func listAgents() {
	apiClient := client.NewClient("")
	
	if err := apiClient.CheckHealth(); err != nil {
		fmt.Println("❌ Server not running")
		os.Exit(1)
	}
	
	result, err := apiClient.ListAgents()
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
	
	count := int(result["count"].(float64))
	fmt.Printf("Deployed Agents: %d\n\n", count)
	
	if agents, ok := result["agents"].([]interface{}); ok {
		for _, a := range agents {
			agent := a.(map[string]interface{})
			fmt.Printf("• %s - %s\n", agent["id"], agent["goal"])
		}
	}
}

func validateSpec(specFile string) {
	fmt.Printf("Validating: %s\n", specFile)

	agentSpec, err := spec.LoadSpec(specFile)
	if err != nil {
		fmt.Printf("❌ Invalid: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Valid!")
	fmt.Printf("   Goal: %s\n", agentSpec.Goal)
	fmt.Printf("   Nodes: %d\n", len(agentSpec.Nodes))
}

func printExecutionResult(result map[string]interface{}) {
	if status, ok := result["status"].(string); ok && status == "error" {
		fmt.Printf("\n❌ Failed: %s\n", result["error"])
		return
	}
	
	fmt.Printf("\n✅ Completed\n")
	
	if cost, ok := result["cost"].(float64); ok {
		fmt.Printf("💰 Cost: $%.4f\n", cost)
	}
	
	if duration, ok := result["duration_ms"].(float64); ok {
		fmt.Printf("⏱️  Time: %.1fs\n", duration/1000)
	}
	
	if output, ok := result["output"].(string); ok {
		fmt.Println("\n📄 Output:")
		fmt.Println("─────────────────────────────────────")
		fmt.Println(output)
		fmt.Println("─────────────────────────────────────")
	}
}

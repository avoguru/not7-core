package main

import (
	"fmt"
	"os"

	"github.com/not7/core/config"
	"github.com/not7/core/executor"
	"github.com/not7/core/server"
	"github.com/not7/core/spec"
)

func main() {
	// Load configuration first
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

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "run":
		// If no file argument, start server mode
		if len(os.Args) < 3 {
			runServer()
		} else {
			// One-off execution with file argument
			runAgent(os.Args[2])
		}
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

func printUsage() {
	fmt.Println(`
NOT7 - Not Your Typical Agent | Agent Runtime

Usage:
  not7 run [agent.json]       Run an agent (one-off if file provided, server if no file)
  not7 validate <agent.json>  Validate a specification file
  not7 --help                 Show this help message

Setup:
  1. Copy not7.conf.example to not7.conf
  2. Edit 'not7.conf' with your OpenAI API key (OpenAI.api_key = sk-...)
  3. Start using NOT7!

One-off Execution:
  not7 run agent.json         Execute agent immediately and exit

Server Mode:
  not7 run                    Start server for production deployment
  
  Server accepts agents via:
    • HTTP API: POST /api/v1/agents/run
    • Deploy folder: Drop JSON files in deploy/ directory

Configuration:
  NOT7_CONFIG                 Path to config file (default: ./not7.conf)
  
  All settings in simple key=value format (not7.conf file):
    OpenAI.api_key                 Your OpenAI API key
    OpenAI.default_model           Model to use (gpt-4, gpt-3.5-turbo, etc.)
    OpenAI.default_temperature     Creativity (0.0-2.0)
    OpenAI.default_max_tokens      Max response length
    Server.port                    HTTP port
    Server.deploy_dir              Deploy folder path
    Server.log_dir                 Logs folder path
    Watcher.poll_interval_seconds  File check frequency

Examples:
  # Development (one-off execution)
  not7 run examples/poem-generator.json

  # Production (server mode)
  not7 run
  
  # Then deploy via HTTP or file drop
  curl -X POST http://localhost:8080/api/v1/agents/run -d @agent.json
  # or
  cp agent.json deploy/
`)
}

func runAgent(specFile string) {
	fmt.Println("NOT7 - Agent Runtime")
	fmt.Println("====================\n")

	// Load spec
	fmt.Printf("📖 Loading spec: %s\n", specFile)
	agentSpec, err := spec.LoadSpec(specFile)
	if err != nil {
		fmt.Printf("❌ Failed to load spec: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Spec loaded successfully\n")

	// Create executor
	exec, err := executor.NewExecutor(agentSpec)
	if err != nil {
		fmt.Printf("❌ Failed to create executor: %v\n", err)
		os.Exit(1)
	}

	// Execute
	output, err := exec.Execute("")
	if err != nil {
		fmt.Printf("❌ Execution failed: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Println("📄 Output:")
	fmt.Println("─────────────────────────────────────")
	fmt.Println(output)
	fmt.Println("─────────────────────────────────────")

	// Save results
	resultFile := specFile + ".result.json"
	if err := spec.SaveSpec(agentSpec, resultFile); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save results: %v\n", err)
	} else {
		fmt.Printf("\n💾 Results saved to: %s\n", resultFile)
	}
}

func validateSpec(specFile string) {
	fmt.Printf("Validating spec: %s\n", specFile)

	agentSpec, err := spec.LoadSpec(specFile)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Spec is valid!")
	fmt.Printf("   Goal: %s\n", agentSpec.Goal)
	fmt.Printf("   Nodes: %d\n", len(agentSpec.Nodes))
	fmt.Printf("   Routes: %d\n", len(agentSpec.Routes))
}

func runServer() {
	cfg := config.Get()
	
	// Create and start server using config values
	srv := server.NewServer(cfg.Server.Port, cfg.Server.DeployDir, cfg.Server.LogDir)
	if err := srv.Start(); err != nil {
		fmt.Printf("❌ Server error: %v\n", err)
		os.Exit(1)
	}
}

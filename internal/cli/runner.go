package cli

import (
	"fmt"
	"os"

	"github.com/not7/core/config"
	"github.com/not7/core/executor"
	"github.com/not7/core/spec"
)

// RunAgentWithTrace executes an agent locally with live trace output
func RunAgentWithTrace(specFile string) error {
	// Load config
	configFile := "not7.conf"
	if envConfig := os.Getenv("NOT7_CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	if _, err := config.LoadConfig(configFile); err != nil {
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	PrintLiveTraceHeader()

	agentSpec, err := spec.LoadSpec(specFile)
	if err != nil {
		return fmt.Errorf("failed to load spec: %w", err)
	}

	fmt.Printf("🎯 Goal: %s\n\n", agentSpec.Goal)

	// Create executor with CLI mode (prints to stdout)
	exec, err := executor.NewExecutor(agentSpec)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Execute
	output, err := exec.Execute("")
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	metadata := exec.GetMetadata()

	// Print final summary
	PrintLiveTraceSummary(metadata, output)

	return nil
}

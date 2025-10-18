package cmd

import (
	"fmt"
	"os"

	"github.com/not7/core/client"
	"github.com/not7/core/internal/cli"
	"github.com/spf13/cobra"
)

var (
	streamMode bool
	asyncMode  bool
)

var runCmd = &cobra.Command{
	Use:   "run <agent.json>",
	Short: "Execute an agent",
	Long:  `Execute an agent from a JSON specification file`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAgent,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVar(&streamMode, "stream", false, "Stream live agent reasoning")
	runCmd.Flags().BoolVar(&asyncMode, "async", false, "Run agent in background")
}

func runAgent(cmd *cobra.Command, args []string) error {
	specFile := args[0]

	// Always use API client (server must be running)
	apiClient := client.NewClient("")

	if err := apiClient.CheckHealth(); err != nil {
		return fmt.Errorf("server not running. Start server first:\n  Terminal 1: ./not7 serve\n  Terminal 2: ./not7 run agent.json")
	}

	agentJSON, err := os.ReadFile(specFile)
	if err != nil {
		return fmt.Errorf("failed to read spec: %w", err)
	}

	fmt.Printf("📖 Executing: %s\n", specFile)

	// Execute via API with stream and async options
	result, err := apiClient.RunAgent(agentJSON, asyncMode, streamMode)
	if err != nil {
		return err
	}

	if asyncMode {
		fmt.Printf("\n✅ Submitted (background)\n")
		fmt.Printf("📋 Execution ID: %s\n\n", result["execution_id"])
		fmt.Printf("Check status: ./not7 status %s\n", result["execution_id"])
	} else {
		cli.PrintExecutionResult(result)
	}

	return nil
}

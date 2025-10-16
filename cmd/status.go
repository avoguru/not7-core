package cmd

import (
	"fmt"

	"github.com/not7/core/client"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <execution-id>",
	Short: "Check execution status",
	Long:  `Check the status of a running or completed agent execution`,
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	execID := args[0]

	apiClient := client.NewClient("")

	if err := apiClient.CheckHealth(); err != nil {
		return fmt.Errorf("server not running")
	}

	status, err := apiClient.GetExecutionStatus(execID)
	if err != nil {
		return err
	}

	fmt.Printf("Execution: %s\n", execID)
	fmt.Printf("Status: %s\n", status["status"])
	fmt.Printf("Goal: %s\n", status["goal"])

	if progress, ok := status["progress"].(map[string]interface{}); ok {
		fmt.Printf("Progress: %v/%v nodes\n",
			progress["completed_nodes"], progress["total_nodes"])
	}

	return nil
}

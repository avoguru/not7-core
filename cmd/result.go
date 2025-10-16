package cmd

import (
	"fmt"

	"github.com/not7/core/client"
	"github.com/not7/core/internal/cli"
	"github.com/spf13/cobra"
)

var resultCmd = &cobra.Command{
	Use:   "result <execution-id>",
	Short: "Get execution result",
	Long:  `Get the result of a completed agent execution`,
	Args:  cobra.ExactArgs(1),
	RunE:  runResult,
}

func init() {
	rootCmd.AddCommand(resultCmd)
}

func runResult(cmd *cobra.Command, args []string) error {
	execID := args[0]

	apiClient := client.NewClient("")

	if err := apiClient.CheckHealth(); err != nil {
		return fmt.Errorf("server not running")
	}

	result, err := apiClient.GetExecutionResult(execID)
	if err != nil {
		return err
	}

	cli.PrintExecutionResult(result)

	return nil
}

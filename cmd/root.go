package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "not7",
	Short: "NOT7 - Agent Runtime",
	Long: `NOT7 - A declarative agent runtime with ReAct reasoning.

NOT7 allows you to define AI agents using JSON specifications and execute
them with built-in chain-of-thought reasoning and tool calling capabilities.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Disable default completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

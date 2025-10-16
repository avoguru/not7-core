package cmd

import (
	"fmt"

	"github.com/not7/core/spec"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <agent.json>",
	Short: "Validate agent specification",
	Long:  `Validate an agent JSON specification file (offline validation)`,
	Args:  cobra.ExactArgs(1),
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	specFile := args[0]

	fmt.Printf("Validating: %s\n", specFile)

	agentSpec, err := spec.LoadSpec(specFile)
	if err != nil {
		return fmt.Errorf("invalid: %w", err)
	}

	fmt.Println("✅ Valid!")
	fmt.Printf("   Goal: %s\n", agentSpec.Goal)
	fmt.Printf("   Nodes: %d\n", len(agentSpec.Nodes))

	return nil
}

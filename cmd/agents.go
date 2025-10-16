package cmd

import (
	"fmt"

	"github.com/not7/core/client"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List deployed agents",
	Long:  `List all agents that have been deployed to the server`,
	RunE:  runAgents,
}

func init() {
	rootCmd.AddCommand(agentsCmd)
}

func runAgents(cmd *cobra.Command, args []string) error {
	apiClient := client.NewClient("")

	if err := apiClient.CheckHealth(); err != nil {
		return fmt.Errorf("server not running")
	}

	result, err := apiClient.ListAgents()
	if err != nil {
		return err
	}

	count := int(result["count"].(float64))
	fmt.Printf("Deployed Agents: %d\n\n", count)

	if agents, ok := result["agents"].([]interface{}); ok {
		for _, a := range agents {
			agent := a.(map[string]interface{})
			fmt.Printf("• %s - %s\n", agent["id"], agent["goal"])
		}
	}

	return nil
}

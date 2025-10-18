package cmd

import (
	"fmt"
	"os"

	"github.com/not7/core/config"
	"github.com/not7/core/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the NOT7 agent server",
	Long:  `Start the NOT7 server to accept and execute agent requests via HTTP API`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	// Load config
	configFile := "not7.conf"
	if envConfig := os.Getenv("NOT7_CONFIG"); envConfig != "" {
		configFile = envConfig
	}

	if _, err := config.LoadConfig(configFile); err != nil {
		return fmt.Errorf("failed to load config from %s: %w\n\nPlease copy not7.conf.example to not7.conf and update with your API key:\n  cp not7.conf.example not7.conf\n  # Then edit not7.conf with your OpenAI API key", configFile, err)
	}

	// Start server
	cfg := config.Get()

	srv := server.NewServer(cfg.Server.Port, cfg.Server.ExecutionsDir, cfg.Server.LogDir)

	if err := srv.Start(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

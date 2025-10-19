package cmd

import (
	"fmt"
	"time"

	"github.com/not7/core/config"
	"github.com/not7/core/tools/arcade"
	"github.com/spf13/cobra"
)

var authorizeCmd = &cobra.Command{
	Use:   "authorize <provider>",
	Short: "Authorize tool provider",
	Long:  `Authorize tool providers like Arcade to access integrated services (Gmail, Slack, etc.)`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthorize,
}

func init() {
	rootCmd.AddCommand(authorizeCmd)
}

func runAuthorize(cmd *cobra.Command, args []string) error {
	provider := args[0]

	if provider != "arcade" {
		return fmt.Errorf("unsupported provider: %s (currently only 'arcade' is supported)", provider)
	}

	// Load config
	configFile := "not7.conf"
	if _, err := config.LoadConfig(configFile); err != nil {
		return fmt.Errorf("failed to load config from %s: %w", configFile, err)
	}

	cfg := config.Get()

	if cfg.Arcade.APIKey == "" {
		return fmt.Errorf("ARCADE_API_KEY not set in not7.conf")
	}

	if cfg.Arcade.UserID == "" {
		return fmt.Errorf("ARCADE_USER_ID not set in not7.conf")
	}

	fmt.Println("🔐 Authorizing Arcade Gmail Tools")
	fmt.Println()

	// Create Arcade client
	client := arcade.NewClient(cfg.Arcade.APIKey, cfg.Arcade.UserID)

	// List Gmail tools to pick one for authorization
	fmt.Println("📋 Fetching Gmail tools...")
	tools, err := client.ListTools("Gmail")
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if len(tools) == 0 {
		return fmt.Errorf("no Gmail tools found")
	}

	// Check if already authorized
	if tools[0].Requirements.Authorization != nil && tools[0].Requirements.Authorization.Status == "active" {
		fmt.Printf("✅ Found %d Gmail tools\n", len(tools))
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("  ✅ Gmail is already authorized!")
		fmt.Println()
		fmt.Println("  You can now use Gmail tools in your agents.")
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		return nil
	}

	// Use the first Gmail tool for authorization
	// All Gmail tools typically share the same OAuth authorization
	firstTool := tools[0].FullyQualifiedName
	fmt.Printf("✅ Found %d Gmail tools\n", len(tools))
	fmt.Printf("🔧 Using tool: %s\n\n", firstTool)

	// Initiate authorization
	fmt.Println("🚀 Initiating OAuth authorization...")
	authResp, err := client.AuthorizeTool(firstTool)
	if err != nil {
		return fmt.Errorf("failed to initiate authorization: %w", err)
	}

	if authResp.Status == "completed" {
		fmt.Println("✅ Authorization already completed!")
		return nil
	}

	if authResp.AuthorizationURL == "" {
		return fmt.Errorf("no authorization URL received")
	}

	// Print authorization URL
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("  Please visit this URL to authorize Gmail access:")
	fmt.Println()
	fmt.Printf("  %s\n", authResp.AuthorizationURL)
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("⏳ Waiting for authorization (timeout: 5 minutes)...")
	fmt.Println()

	// Poll for authorization status with 5-minute timeout
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("authorization timeout (5 minutes)")
		case <-ticker.C:
			// Check status with long-polling (wait up to 30 seconds for status change)
			statusResp, err := client.CheckAuthStatus(authResp.AuthorizationID, 30)
			if err != nil {
				fmt.Printf("⚠️  Error checking status: %v\n", err)
				continue
			}

			if statusResp.Status == "completed" {
				fmt.Println()
				fmt.Println("✅ Authorization completed successfully!")
				fmt.Println()
				fmt.Println("You can now use Gmail tools in your agents.")
				return nil
			}

			if statusResp.Status == "failed" {
				errorMsg := "unknown error"
				if statusResp.Error != "" {
					errorMsg = statusResp.Error
				}
				return fmt.Errorf("authorization failed: %s", errorMsg)
			}

			// Still pending, continue polling
		}
	}
}

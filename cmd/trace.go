package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/not7/core/internal/cli"
	"github.com/not7/core/spec"
	"github.com/spf13/cobra"
)

var traceCmd = &cobra.Command{
	Use:   "trace [agent-id]",
	Short: "View detailed ReAct execution trace",
	Long:  `Display the chain of thought and tool calls from the last execution of an agent`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTrace,
}

func init() {
	rootCmd.AddCommand(traceCmd)
	traceCmd.Flags().StringP("file", "f", "", "Specific agent JSON file to view trace for")
	traceCmd.Flags().BoolP("full", "F", false, "Show full thoughts (not truncated)")
}

func runTrace(cmd *cobra.Command, args []string) error {
	filePath, _ := cmd.Flags().GetString("file")
	showFull, _ := cmd.Flags().GetBool("full")

	// Find most recent log file
	logsDir := "./logs"
	files, err := os.ReadDir(logsDir)
	if err != nil {
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	// Filter to trace JSON files and sort by modification time
	var jsonFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), "-trace.json") {
			jsonFiles = append(jsonFiles, filepath.Join(logsDir, file.Name()))
		}
	}

	if len(jsonFiles) == 0 {
		return fmt.Errorf("no execution traces found in %s", logsDir)
	}

	// Sort by modified time (most recent first)
	sort.Slice(jsonFiles, func(i, j int) bool {
		infoI, _ := os.Stat(jsonFiles[i])
		infoJ, _ := os.Stat(jsonFiles[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	traceFile := jsonFiles[0]
	if filePath != "" {
		traceFile = filePath
	}

	// Read trace file
	data, err := os.ReadFile(traceFile)
	if err != nil {
		return fmt.Errorf("failed to read trace file: %w", err)
	}

	var agentSpec spec.AgentSpec
	if err := json.Unmarshal(data, &agentSpec); err != nil {
		return fmt.Errorf("failed to parse trace: %w", err)
	}

	// Display trace
	cli.DisplayTrace(&agentSpec, showFull)

	return nil
}

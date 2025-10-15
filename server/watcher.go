package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/not7/core/config"
)

// watchDeployFolder watches the deploy directory for new JSON files
// Files are automatically executed and moved to processed/ folder
func (s *Server) watchDeployFolder() {
	cfg := config.Get()
	pollInterval := time.Duration(cfg.Watcher.PollIntervalSeconds) * time.Second

	fmt.Printf("[File Watcher] Started watching: %s\n", s.deployDir)
	fmt.Printf("[File Watcher] Poll interval: %v\n", pollInterval)

	for {
		files, err := os.ReadDir(s.deployDir)
		if err != nil {
			fmt.Printf("[File Watcher] Error reading deploy directory: %v\n", err)
			time.Sleep(pollInterval)
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			filePath := filepath.Join(s.deployDir, file.Name())
			fmt.Printf("[File Watcher] Detected new agent: %s\n", file.Name())

			// Execute the agent
			s.executeAgentFromFile(filePath)

			// Move to processed folder
			processedPath := filepath.Join(s.deployDir, "processed", file.Name())
			if err := os.Rename(filePath, processedPath); err != nil {
				fmt.Printf("[File Watcher] Warning: Failed to move file to processed: %v\n", err)
			} else {
				fmt.Printf("[File Watcher] Moved to processed: %s\n", file.Name())
			}
		}

		time.Sleep(pollInterval)
	}
}

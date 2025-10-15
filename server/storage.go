package server

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/not7/core/spec"
)

// saveAgentSpec saves an agent specification to the specs directory
func (s *Server) saveAgentSpec(id string, agentSpec *spec.AgentSpec) error {
	specPath := filepath.Join(s.deployDir, "specs", id+".json")
	return spec.SaveSpec(agentSpec, specPath)
}

// loadAgentSpec loads an agent specification from the specs directory
func (s *Server) loadAgentSpec(id string) (*spec.AgentSpec, error) {
	specPath := filepath.Join(s.deployDir, "specs", id+".json")
	return spec.LoadSpec(specPath)
}

// deleteAgentSpec deletes an agent specification from the specs directory
func (s *Server) deleteAgentSpec(id string) error {
	specPath := filepath.Join(s.deployDir, "specs", id+".json")
	return os.Remove(specPath)
}

// agentExists checks if an agent specification file exists
func (s *Server) agentExists(id string) bool {
	specPath := filepath.Join(s.deployDir, "specs", id+".json")
	_, err := os.Stat(specPath)
	return err == nil
}

// listAgentSpecs returns a list of all deployed agent specifications
func (s *Server) listAgentSpecs() ([]AgentInfo, error) {
	specsDir := filepath.Join(s.deployDir, "specs")
	files, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, err
	}

	var agents []AgentInfo
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(file.Name(), ".json")
		agentSpec, err := s.loadAgentSpec(id)
		if err != nil {
			continue
		}

		info, _ := file.Info()
		agents = append(agents, AgentInfo{
			ID:        agentSpec.ID,
			Goal:      agentSpec.Goal,
			CreatedAt: info.ModTime().Format(time.RFC3339),
		})
	}

	return agents, nil
}

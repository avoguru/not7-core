package spec

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadSpec loads and parses a NOT7 agent specification from a JSON file
func LoadSpec(filepath string) (*AgentSpec, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var spec AgentSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse spec JSON: %w", err)
	}

	// Validate spec
	if err := ValidateSpec(&spec); err != nil {
		return nil, fmt.Errorf("invalid spec: %w", err)
	}

	return &spec, nil
}

// ValidateSpec ensures the spec is valid
func ValidateSpec(spec *AgentSpec) error {
	if spec.Version == "" {
		return fmt.Errorf("version is required")
	}
	if spec.Goal == "" {
		return fmt.Errorf("goal is required")
	}
	if len(spec.Nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}
	if len(spec.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
	}

	// Validate nodes
	nodeIDs := make(map[string]bool)
	for _, node := range spec.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID is required")
		}
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		if node.Type == "" {
			return fmt.Errorf("node type is required for node %s", node.ID)
		}
		if node.Type == "llm" && node.Prompt == "" {
			return fmt.Errorf("prompt is required for LLM node %s", node.ID)
		}
	}

	// Validate routes
	for _, route := range spec.Routes {
		if route.From == "" || route.To == "" {
			return fmt.Errorf("route must have both 'from' and 'to'")
		}
		// Check that referenced nodes exist (except start/end)
		if route.From != "start" && !nodeIDs[route.From] {
			return fmt.Errorf("route references unknown node: %s", route.From)
		}
		if route.To != "end" && !nodeIDs[route.To] {
			return fmt.Errorf("route references unknown node: %s", route.To)
		}
	}

	return nil
}

// SaveSpec saves a spec to a JSON file
func SaveSpec(spec *AgentSpec, filepath string) error {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal spec: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write spec file: %w", err)
	}

	return nil
}


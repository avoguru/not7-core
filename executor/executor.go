package executor

import (
	"fmt"
	"time"

	"github.com/not7/core/llm"
	"github.com/not7/core/logger"
	"github.com/not7/core/spec"
)

// Logger interface for logging
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// Executor runs an agent specification
type Executor struct {
	spec      *spec.AgentSpec
	llmClient *llm.OpenAIClient
	nodeMap   map[string]*spec.Node
	results   map[string]*spec.NodeResult
	logger    Logger
	useCLI    bool // Flag to determine if we should print to stdout
}

// NewExecutor creates a new executor for CLI mode (prints to stdout)
func NewExecutor(agentSpec *spec.AgentSpec) (*Executor, error) {
	return newExecutor(agentSpec, logger.NewConsoleLogger(), true)
}

// NewExecutorWithLogger creates a new executor with a custom logger (for server mode)
func NewExecutorWithLogger(agentSpec *spec.AgentSpec, log Logger) (*Executor, error) {
	return newExecutor(agentSpec, log, false)
}

// newExecutor is the internal constructor
func newExecutor(agentSpec *spec.AgentSpec, log Logger, useCLI bool) (*Executor, error) {
	llmClient, err := llm.NewOpenAIClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Build node map for quick lookup
	nodeMap := make(map[string]*spec.Node)
	for i := range agentSpec.Nodes {
		nodeMap[agentSpec.Nodes[i].ID] = &agentSpec.Nodes[i]
	}

	return &Executor{
		spec:      agentSpec,
		llmClient: llmClient,
		nodeMap:   nodeMap,
		results:   make(map[string]*spec.NodeResult),
		logger:    log,
		useCLI:    useCLI,
	}, nil
}

// Execute runs the agent
func (e *Executor) Execute(input string) (string, error) {
	startTime := time.Now()

	// Log start
	e.logger.Info("Starting agent: %s", e.spec.Goal)
	e.logger.Info("Version: %s", e.spec.Version)

	// Print to stdout if CLI mode
	if e.useCLI {
		fmt.Printf("🚀 Starting agent: %s\n", e.spec.Goal)
		fmt.Printf("📋 Version: %s\n\n", e.spec.Version)
	}

	// Initialize metadata
	if e.spec.Metadata == nil {
		e.spec.Metadata = &spec.Metadata{}
	}
	e.spec.Metadata.ExecutedAt = time.Now().Format(time.RFC3339)
	e.spec.Metadata.Status = "running"

	// Find starting nodes (routes from "start")
	startingNodes := e.findNodesFrom("start")
	if len(startingNodes) == 0 {
		return "", fmt.Errorf("no routes from 'start' found")
	}

	// Execute starting nodes
	currentOutput := input
	for _, nodeID := range startingNodes {
		output, err := e.executeNode(nodeID, currentOutput)
		if err != nil {
			e.spec.Metadata.Status = "failed"
			e.logger.Error("Execution failed at node %s: %v", nodeID, err)
			return "", fmt.Errorf("execution failed at node %s: %w", nodeID, err)
		}
		currentOutput = output

		// Follow routes from this node
		nextOutput, err := e.followRoutes(nodeID, currentOutput)
		if err != nil {
			e.spec.Metadata.Status = "failed"
			e.logger.Error("Routing failed: %v", err)
			return "", fmt.Errorf("routing failed: %w", err)
		}
		currentOutput = nextOutput
	}

	// Update metadata
	e.spec.Metadata.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	e.spec.Metadata.Status = "success"

	// Calculate total cost
	totalCost := 0.0
	for _, result := range e.results {
		totalCost += result.Cost
	}
	e.spec.Metadata.TotalCost = totalCost

	// Convert results map to slice
	var nodeResults []spec.NodeResult
	for _, result := range e.results {
		nodeResults = append(nodeResults, *result)
	}
	e.spec.Metadata.NodeResults = nodeResults

	// Log completion
	e.logger.Info("Execution completed successfully in %dms", e.spec.Metadata.ExecutionTimeMs)
	e.logger.Info("Total cost: $%.4f", totalCost)

	// Print to stdout if CLI mode
	if e.useCLI {
		fmt.Printf("\n✅ Execution completed in %dms\n", e.spec.Metadata.ExecutionTimeMs)
		fmt.Printf("💰 Total cost: $%.4f\n\n", totalCost)
	}

	return currentOutput, nil
}

// executeNode executes a single node
func (e *Executor) executeNode(nodeID string, input string) (string, error) {
	node := e.nodeMap[nodeID]
	if node == nil {
		return "", fmt.Errorf("node not found: %s", nodeID)
	}

	// Log node execution
	e.logger.Info("Executing node: %s (%s)", node.Name, node.Type)

	// Print to stdout if CLI mode
	if e.useCLI {
		fmt.Printf("⚙️  Executing node: %s (%s)\n", node.Name, node.Type)
	}

	startTime := time.Now()

	result := &spec.NodeResult{
		NodeID: nodeID,
		Input:  input,
		Status: "running",
	}

	var output string
	var cost float64
	var err error

	switch node.Type {
	case "llm":
		output, cost, err = e.executeLLMNode(node, input)
	default:
		err = fmt.Errorf("unsupported node type: %s", node.Type)
	}

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	result.Cost = cost

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		e.results[nodeID] = result
		e.logger.Error("Node %s failed: %v", nodeID, err)
		return "", err
	}

	result.Status = "success"
	result.Output = output
	e.results[nodeID] = result

	// Log completion
	e.logger.Info("Node %s completed in %dms (cost: $%.4f)", nodeID, result.ExecutionTimeMs, cost)

	// Print to stdout if CLI mode
	if e.useCLI {
		fmt.Printf("   ✓ Completed in %dms (cost: $%.4f)\n", result.ExecutionTimeMs, cost)
	}

	return output, nil
}

// executeLLMNode executes an LLM node
func (e *Executor) executeLLMNode(node *spec.Node, input string) (string, float64, error) {
	// Determine LLM config (node-specific or global)
	llmConfig := node.LLM
	if llmConfig == nil && e.spec.Config != nil {
		llmConfig = e.spec.Config.LLM
	}
	if llmConfig == nil {
		return "", 0, fmt.Errorf("no LLM configuration found")
	}

	// Set defaults
	if llmConfig.Model == "" {
		llmConfig.Model = "gpt-3.5-turbo"
	}
	if llmConfig.Temperature == 0 {
		llmConfig.Temperature = 0.7
	}

	// Execute
	output, cost, err := e.llmClient.Execute(llmConfig, node.Prompt, input)
	if err != nil {
		return "", 0, err
	}

	return output, cost, nil
}

// followRoutes follows routes from a node
func (e *Executor) followRoutes(fromNodeID string, input string) (string, error) {
	nextNodes := e.findNodesFrom(fromNodeID)
	if len(nextNodes) == 0 {
		// No more routes, we're done
		return input, nil
	}

	// For now, just follow the first route (simple linear execution)
	// TODO: Handle parallel execution, conditionals, etc.
	currentOutput := input
	for _, nodeID := range nextNodes {
		if nodeID == "end" {
			return currentOutput, nil
		}

		output, err := e.executeNode(nodeID, currentOutput)
		if err != nil {
			return "", err
		}
		currentOutput = output

		// Recursively follow routes
		nextOutput, err := e.followRoutes(nodeID, currentOutput)
		if err != nil {
			return "", err
		}
		currentOutput = nextOutput
	}

	return currentOutput, nil
}

// findNodesFrom finds nodes that are routed from the given node
func (e *Executor) findNodesFrom(fromNodeID string) []string {
	var nodes []string
	for _, route := range e.spec.Routes {
		if route.From == fromNodeID {
			nodes = append(nodes, route.To)
		}
	}
	return nodes
}

// GetMetadata returns execution metadata
func (e *Executor) GetMetadata() *spec.Metadata {
	return e.spec.Metadata
}

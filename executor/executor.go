package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/not7/core/config"
	"github.com/not7/core/llm"
	"github.com/not7/core/logger"
	"github.com/not7/core/spec"
	"github.com/not7/core/tools"
	"github.com/not7/core/tools/arcade"
	"github.com/not7/core/tools/builtin"
)

// Logger interface for logging
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// Executor runs an agent specification
type Executor struct {
	spec         *spec.AgentSpec
	llmClient    *llm.OpenAIClient
	nodeMap      map[string]*spec.Node
	results      map[string]*spec.NodeResult
	logger       Logger
	useCLI       bool                        // Flag to determine if we should print to stdout
	toolManagers map[string]*tools.Manager // Pool of tool managers by provider
	cfg          *config.Config              // Global config for tool initialization
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

	// Get global config
	cfg := config.Get()

	// Create executor with tool manager pool
	executor := &Executor{
		spec:         agentSpec,
		llmClient:    llmClient,
		nodeMap:      nodeMap,
		results:      make(map[string]*spec.NodeResult),
		logger:       log,
		useCLI:       useCLI,
		toolManagers: make(map[string]*tools.Manager),
		cfg:          cfg,
	}

	// Initialize default tool manager if agent-level tools are configured
	if agentSpec.Config != nil && agentSpec.Config.Tools != nil {
		provider := agentSpec.Config.Tools.Provider
		_, err := executor.getOrCreateToolManager(provider)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize default tool manager: %w", err)
		}
	}

	return executor, nil
}

// getOrCreateToolManager returns a tool manager for the given provider,
// creating and initializing it if it doesn't exist in the pool
func (e *Executor) getOrCreateToolManager(provider string) (*tools.Manager, error) {
	// Check if already exists in pool
	if mgr, exists := e.toolManagers[provider]; exists {
		return mgr, nil
	}

	// Create new tool manager
	toolMgr := tools.NewManager("")

	// Initialize based on provider type
	if provider == "builtin" {
		if e.cfg.Builtin.SerpAPIKey == "" {
			return nil, fmt.Errorf("builtin provider requires SERP_API_KEY in not7.conf")
		}

		builtinProvider := builtin.NewProvider(e.cfg.Builtin.SerpAPIKey)
		providerConfig := map[string]string{
			"serp_api_key": e.cfg.Builtin.SerpAPIKey,
		}

		if err := builtinProvider.Initialize(providerConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize builtin provider: %w", err)
		}

		if err := toolMgr.RegisterProvider(builtinProvider); err != nil {
			return nil, fmt.Errorf("failed to register builtin provider: %w", err)
		}

		e.logger.Info("Builtin tool provider initialized with %d tools", len(toolMgr.ListTools()))
	} else if provider == "arcade" || (len(provider) > 7 && provider[:7] == "arcade-") {
		// Arcade provider (supports arcade-{toolkit} pattern)
		if e.cfg.Arcade.APIKey == "" {
			return nil, fmt.Errorf("ARCADE_API_KEY not configured in not7.conf")
		}
		if e.cfg.Arcade.UserID == "" {
			return nil, fmt.Errorf("ARCADE_USER_ID not configured in not7.conf")
		}

		// Extract toolkit name from provider (e.g., "arcade-spotify" → "Spotify")
		toolkit := "Gmail" // Default for backward compatibility with "arcade"
		if len(provider) > 7 && provider[:7] == "arcade-" {
			toolkit = provider[7:]
			// Capitalize first letter for API compatibility
			if len(toolkit) > 0 {
				toolkit = strings.ToUpper(toolkit[:1]) + toolkit[1:]
			}
		}

		arcadeProvider := arcade.NewProvider(e.cfg.Arcade.APIKey, e.cfg.Arcade.UserID, toolkit)
		providerConfig := map[string]string{
			"arcade_api_key": e.cfg.Arcade.APIKey,
			"arcade_user_id": e.cfg.Arcade.UserID,
		}

		if err := arcadeProvider.Initialize(providerConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize arcade provider: %w", err)
		}

		// Check authorization status and handle interactive auth if needed
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
		defer cancel()

		if err := arcadeProvider.CheckAndHandleAuthorization(ctx); err != nil {
			return nil, fmt.Errorf("arcade authorization failed: %w", err)
		}

		if err := toolMgr.RegisterProvider(arcadeProvider); err != nil {
			return nil, fmt.Errorf("failed to register arcade provider: %w", err)
		}

		e.logger.Info("Arcade tool provider initialized with %d %s tools", len(toolMgr.ListTools()), toolkit)
	} else {
		return nil, fmt.Errorf("unsupported tool provider: %s", provider)
	}

	// Store in pool
	e.toolManagers[provider] = toolMgr
	return toolMgr, nil
}

// getToolManagerForNode resolves and returns the appropriate tool manager for a node
func (e *Executor) getToolManagerForNode(node *spec.Node) (*tools.Manager, error) {
	// Check node-level config first (highest priority)
	if node.Config != nil && node.Config.Tools != nil {
		provider := node.Config.Tools.Provider
		return e.getOrCreateToolManager(provider)
	}

	// Fall back to agent-level config
	if e.spec.Config != nil && e.spec.Config.Tools != nil {
		provider := e.spec.Config.Tools.Provider
		// Should already exist from initialization, but get or create just in case
		return e.getOrCreateToolManager(provider)
	}

	// No tools configured
	return nil, nil
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
	var reactTrace *spec.ReActTrace

	switch node.Type {
	case "llm":
		output, cost, err = e.executeLLMNode(node, input)
	case "react":
		// Check if tools are enabled for this node
		if node.ToolsEnabled {
			// Resolve tool manager for this node
			toolMgr, toolErr := e.getToolManagerForNode(node)
			if toolErr != nil {
				err = fmt.Errorf("failed to get tool manager: %w", toolErr)
			} else if toolMgr != nil && toolMgr.HasTools() {
				output, cost, reactTrace, err = e.executeReActNodeWithTools(node, input, toolMgr)
			} else {
				output, cost, reactTrace, err = e.executeReActNode(node, input)
			}
		} else {
			output, cost, reactTrace, err = e.executeReActNode(node, input)
		}
	case "tool":
		output, cost, err = e.executeToolNode(node, input)
	default:
		err = fmt.Errorf("unsupported node type: %s", node.Type)
	}
	result.Cost = cost

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	result.ReActTrace = reactTrace

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

// executeToolNode executes an explicit tool node
func (e *Executor) executeToolNode(node *spec.Node, input string) (string, float64, error) {
	// Resolve tool manager for this node
	toolMgr, err := e.getToolManagerForNode(node)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get tool manager: %w", err)
	}
	if toolMgr == nil {
		return "", 0, fmt.Errorf("tool manager not initialized - tools not configured")
	}

	if node.ToolName == "" {
		return "", 0, fmt.Errorf("tool_name is required for tool nodes")
	}

	e.logger.Info("Executing tool: %s", node.ToolName)

	// Prepare arguments
	args := node.ToolArguments
	if args == nil {
		args = make(map[string]interface{})
	}

	// Support {{input}} placeholder in arguments
	for key, val := range args {
		if strVal, ok := val.(string); ok && strVal == "{{input}}" {
			args[key] = input
		}
	}

	// Execute tool
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := toolMgr.ExecuteTool(ctx, node.ToolName, args)
	if err != nil {
		return "", 0, fmt.Errorf("tool execution failed: %w", err)
	}

	if !result.Success {
		return "", 0, fmt.Errorf("tool returned error: %s", result.Error)
	}

	// Convert output to string
	output := fmt.Sprintf("%v", result.Output)
	return output, 0, nil
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

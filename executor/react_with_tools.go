package executor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/not7/core/spec"
	"github.com/not7/core/tools"
)

// parseToolCall extracts tool calls from LLM response
// Format: TOOL_CALL: tool_name\n{json_arguments}
func parseToolCall(response string) (string, map[string]interface{}, bool) {
	// Pattern: TOOL_CALL: tool_name
	re := regexp.MustCompile(`(?m)^TOOL_CALL:\s*(\S+)\s*$`)
	matches := re.FindStringSubmatch(response)

	if len(matches) < 2 {
		return "", nil, false
	}

	toolName := strings.TrimSpace(matches[1])

	// Find JSON block after TOOL_CALL line
	lines := strings.Split(response, "\n")
	jsonStart := -1
	for i, line := range lines {
		if strings.Contains(line, "TOOL_CALL:") {
			jsonStart = i + 1
			break
		}
	}

	if jsonStart == -1 || jsonStart >= len(lines) {
		// No JSON provided, return empty args
		return toolName, make(map[string]interface{}), true
	}

	// Try to parse JSON from remaining lines
	jsonText := strings.Join(lines[jsonStart:], "\n")
	jsonText = strings.TrimSpace(jsonText)

	// Simple JSON parsing - look for {...} block
	braceStart := strings.Index(jsonText, "{")
	if braceStart == -1 {
		return toolName, make(map[string]interface{}), true
	}

	// Find matching closing brace
	braceCount := 0
	braceEnd := -1
	for i := braceStart; i < len(jsonText); i++ {
		if jsonText[i] == '{' {
			braceCount++
		} else if jsonText[i] == '}' {
			braceCount--
			if braceCount == 0 {
				braceEnd = i + 1
				break
			}
		}
	}

	if braceEnd == -1 {
		return toolName, make(map[string]interface{}), true
	}

	jsonBlock := jsonText[braceStart:braceEnd]

	// Parse JSON (simplified - you might want to use encoding/json here)
	// For now, return empty map and let the tool provider handle it
	args := make(map[string]interface{})

	// Quick and dirty JSON parsing for simple cases
	// Remove braces
	content := strings.Trim(jsonBlock, "{}")
	content = strings.TrimSpace(content)

	if content != "" {
		// Split by commas (naive, doesn't handle nested objects)
		pairs := strings.Split(content, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				key := strings.Trim(strings.TrimSpace(kv[0]), "\"")
				value := strings.TrimSpace(kv[1])
				value = strings.Trim(value, "\"")
				args[key] = value
			}
		}
	}

	return toolName, args, true
}

// executeReActNodeWithTools executes a ReAct node with tool calling support
func (e *Executor) executeReActNodeWithTools(node *spec.Node, input string, toolMgr *tools.Manager) (string, float64, *spec.ReActTrace, error) {
	// Get LLM config
	llmConfig := node.LLM
	if llmConfig == nil && e.spec.Config != nil {
		llmConfig = e.spec.Config.LLM
	}
	if llmConfig == nil {
		return "", 0, nil, fmt.Errorf("no LLM configuration found")
	}

	maxIterations := node.MaxIterations
	if maxIterations == 0 {
		maxIterations = 5
	}

	// Build system prompt with tool context
	systemPrompt := e.buildReActSystemPromptWithTools(node.ReActGoal, node.ThinkingPrompt, toolMgr)

	e.logger.Info("Starting ReAct reasoning with tools (max iterations: %d)", maxIterations)
	e.logger.Info("Available tools: %d", len(toolMgr.ListTools()))

	if e.useCLI {
		fmt.Printf("   🧠 ReAct Goal: %s\n", node.ReActGoal)
		fmt.Printf("   🔄 Max iterations: %d\n", maxIterations)
		fmt.Printf("   🛠️  Tools available: %d\n\n", len(toolMgr.ListTools()))
	}

	// Initialize trace
	trace := &spec.ReActTrace{
		ThinkingSteps: make([]spec.ThinkingStep, 0),
	}

	totalCost := 0.0
	startTime := time.Now()
	var finalAnswer string
	conversationContext := ""

	// Iteration loop
	for i := 1; i <= maxIterations; i++ {
		iterStart := time.Now()

		e.logger.Info("ReAct iteration %d/%d", i, maxIterations)
		if e.useCLI {
			fmt.Printf("   💭 Iteration %d/%d\n", i, maxIterations)
		}

		// Build prompt for this iteration
		var iterationPrompt string
		if i == 1 {
			iterationPrompt = fmt.Sprintf("Goal: %s\n\nYou have access to tools. Use them to help achieve the goal.\n\nBegin your reasoning.", node.ReActGoal)
		} else {
			iterationPrompt = fmt.Sprintf("%s\n\nContinue your reasoning. You can:\n1. Call a tool using TOOL_CALL: tool_name format\n2. Finish with FINAL: your_answer", conversationContext)
		}

		// Execute LLM call
		response, cost, err := e.llmClient.Execute(llmConfig, systemPrompt, iterationPrompt)
		if err != nil {
			e.logger.Error("ReAct iteration %d failed: %v", i, err)
			return "", totalCost, trace, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		iterDuration := time.Since(iterStart).Milliseconds()
		totalCost += cost

		// Initialize thinking step
		step := spec.ThinkingStep{
			Iteration:  i,
			Thought:    response,
			DurationMs: iterDuration,
			Cost:       cost,
			ToolCalls:  make([]spec.ToolCallTrace, 0),
		}

		e.logger.Info("Iteration %d LLM response received (cost: $%.4f)", i, cost)

		// Check for tool call
		toolName, args, hasTool := parseToolCall(response)
		if hasTool {
			e.logger.Info("Tool call detected: %s", toolName)
			if e.useCLI {
				fmt.Printf("      🔧 Calling tool: %s\n", toolName)
			}

			// Execute tool
			toolStart := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			toolResult, toolErr := toolMgr.ExecuteTool(ctx, toolName, args)
			toolDuration := time.Since(toolStart).Milliseconds()

			// Record tool call
			toolTrace := spec.ToolCallTrace{
				ToolName:   toolName,
				Arguments:  args,
				DurationMs: toolDuration,
			}

			if toolErr != nil {
				toolTrace.Error = toolErr.Error()
				e.logger.Error("Tool execution failed: %v", toolErr)

				// Add error to context
				conversationContext += fmt.Sprintf("\n\nTOOL_RESULT (%s): ERROR - %s", toolName, toolErr.Error())
			} else {
				toolTrace.Result = toolResult.Output
				e.logger.Info("Tool executed successfully in %dms", toolDuration)

				if e.useCLI {
					fmt.Printf("         ✓ Tool completed in %dms\n", toolDuration)
				}

				// Add result to context
				resultStr := fmt.Sprintf("%v", toolResult.Output)
				if len(resultStr) > 500 {
					resultStr = resultStr[:500] + "... (truncated)"
				}
				conversationContext += fmt.Sprintf("\n\nTOOL_RESULT (%s):\n%s", toolName, resultStr)
			}

			step.ToolCalls = append(step.ToolCalls, toolTrace)
		} else {
			// No tool call, add thought to context
			conversationContext += fmt.Sprintf("\n\n%s", response)
		}

		// Add step to trace
		trace.ThinkingSteps = append(trace.ThinkingSteps, step)

		if e.useCLI {
			preview := getThoughtPreview(response)
			fmt.Printf("      %s\n", preview)
			fmt.Printf("      ⏱️  %dms | 💰 $%.4f\n\n", iterDuration, cost)
		}

		// Check if final answer
		if strings.HasPrefix(strings.TrimSpace(response), "FINAL:") {
			finalAnswer = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(response), "FINAL:"))
			e.logger.Info("ReAct reached conclusion at iteration %d", i)
			if e.useCLI {
				fmt.Printf("   ✅ Conclusion reached at iteration %d\n\n", i)
			}
			break
		}

		finalAnswer = response // Keep latest as final if we don't get FINAL: marker
	}

	// Finalize trace
	trace.Iterations = len(trace.ThinkingSteps)
	trace.TotalThinkingTimeMs = time.Since(startTime).Milliseconds()
	trace.IterationsCost = totalCost

	e.logger.Info("ReAct complete: %d iterations, %dms total, $%.4f cost",
		trace.Iterations, trace.TotalThinkingTimeMs, totalCost)

	return finalAnswer, totalCost, trace, nil
}

// buildReActSystemPromptWithTools creates the system prompt including tool descriptions
func (e *Executor) buildReActSystemPromptWithTools(goal, customThinking string, toolMgr *tools.Manager) string {
	thinkingGuidance := customThinking
	if thinkingGuidance == "" {
		thinkingGuidance = `Process:
1. THINK: What do you currently know? What's missing? What tools can help?
2. ACT: Call tools to gather information using the TOOL_CALL format
3. OBSERVE: Review tool results and integrate them into your understanding
4. REASON: Based on your thinking and tool results, what's your current best answer?
5. CRITIQUE: Is your answer complete and accurate? Do you need more information?

To call a tool, use this exact format:
TOOL_CALL: tool_name
{
  "argument1": "value1",
  "argument2": "value2"
}

If your answer is satisfactory and complete, start your response with "FINAL:" followed by your final answer.
If you need more thinking or tool calls, continue reasoning.`
	}

	// Get tool context
	toolContext := toolMgr.GetToolContext()

	return fmt.Sprintf(`You are a research and reasoning assistant with access to tools.

Your goal: %s

%s

%s

Iterate and refine your thinking until you have a complete, accurate answer.`,
		goal, toolContext, thinkingGuidance)
}

package executor

import (
	"fmt"
	"strings"
	"time"

	"github.com/not7/core/config"
	"github.com/not7/core/spec"
)

// executeReActNode executes a ReAct (Reasoning + Acting) node with iterative thinking
func (e *Executor) executeReActNode(node *spec.Node, input string) (string, float64, *spec.ReActTrace, error) {
	cfg := config.Get()

	// Get LLM config
	llmConfig := node.LLM
	if llmConfig == nil && e.spec.Config != nil {
		llmConfig = e.spec.Config.LLM
	}
	if llmConfig == nil {
		return "", 0, nil, fmt.Errorf("no LLM configuration found")
	}

	// Set defaults
	if llmConfig.Model == "" {
		llmConfig.Model = cfg.OpenAI.DefaultModel
	}
	if llmConfig.Temperature == 0 {
		llmConfig.Temperature = cfg.OpenAI.DefaultTemperature
	}

	maxIterations := node.MaxIterations
	if maxIterations == 0 {
		maxIterations = 5 // Default to 5 iterations
	}

	// Build system prompt for ReAct
	systemPrompt := buildReActSystemPrompt(node.ReActGoal, node.ThinkingPrompt)

	e.logger.Info("Starting ReAct reasoning (max iterations: %d)", maxIterations)
	if e.useCLI {
		fmt.Printf("   🧠 ReAct Goal: %s\n", node.ReActGoal)
		fmt.Printf("   🔄 Max iterations: %d\n\n", maxIterations)
	}

	// Initialize trace
	trace := &spec.ReActTrace{
		ThinkingSteps: make([]spec.ThinkingStep, 0),
	}

	totalCost := 0.0
	startTime := time.Now()
	var finalAnswer string

	// Iteration loop
	for i := 1; i <= maxIterations; i++ {
		iterStart := time.Now()

		e.logger.Info("ReAct iteration %d/%d", i, maxIterations)
		if e.useCLI {
			fmt.Printf("   💭 Iteration %d/%d: Thinking...\n", i, maxIterations)
		}

		// Build prompt for this iteration
		var iterationPrompt string
		if i == 1 {
			iterationPrompt = fmt.Sprintf("Goal: %s\n\nBegin your reasoning. Think step by step.", node.ReActGoal)
		} else {
			iterationPrompt = "Continue your reasoning. Critique your previous thoughts and refine your answer. If you have a complete answer, start with 'FINAL:'"
		}

		// Execute LLM call
		response, cost, err := e.llmClient.Execute(llmConfig, systemPrompt, iterationPrompt)
		if err != nil {
			e.logger.Error("ReAct iteration %d failed: %v", i, err)
			return "", totalCost, trace, fmt.Errorf("iteration %d failed: %w", i, err)
		}

		iterDuration := time.Since(iterStart).Milliseconds()
		totalCost += cost

		// Record this thinking step
		step := spec.ThinkingStep{
			Iteration:  i,
			Thought:    response,
			DurationMs: iterDuration,
			Cost:       cost,
		}
		trace.ThinkingSteps = append(trace.ThinkingSteps, step)

		e.logger.Info("Iteration %d completed in %dms (cost: $%.4f)", i, iterDuration, cost)
		if e.useCLI {
			// Show preview of thought
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

// buildReActSystemPrompt creates the system prompt for ReAct reasoning
func buildReActSystemPrompt(goal, customThinking string) string {
	thinkingGuidance := customThinking
	if thinkingGuidance == "" {
		thinkingGuidance = `Process:
1. THINK: What do you currently know? What's missing?
2. REASON: What should you explore next?
3. ANSWER: Based on your thinking, what's your current best answer?
4. CRITIQUE: Is your answer complete and accurate? What could be improved?

If your answer is satisfactory and complete, start your response with "FINAL:" followed by your final answer.
If you need more thinking, start with "CONTINUE:" and continue reasoning.`
	}

	return fmt.Sprintf(`You are a research and reasoning assistant.

Your goal: %s

%s

Iterate and refine your thinking until you have a complete, accurate answer.`, goal, thinkingGuidance)
}

// getThoughtPreview returns a short preview of the thought for display
func getThoughtPreview(thought string) string {
	lines := strings.Split(thought, "\n")
	if len(lines) == 0 {
		return ""
	}

	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > 80 {
		return firstLine[:77] + "..."
	}
	return firstLine
}

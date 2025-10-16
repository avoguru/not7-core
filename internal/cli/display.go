package cli

import (
	"fmt"
	"strings"

	"github.com/not7/core/spec"
)

// PrintExecutionResult prints the result of an agent execution
func PrintExecutionResult(result map[string]interface{}) {
	if status, ok := result["status"].(string); ok && status == "error" {
		fmt.Printf("\n❌ Failed: %s\n", result["error"])
		return
	}

	fmt.Printf("\n✅ Completed\n")

	if cost, ok := result["cost"].(float64); ok {
		fmt.Printf("💰 Cost: $%.4f\n", cost)
	}

	if duration, ok := result["duration_ms"].(float64); ok {
		fmt.Printf("⏱️  Time: %.1fs\n", duration/1000)
	}

	if output, ok := result["output"].(string); ok {
		fmt.Println("\n📄 Output:")
		fmt.Println("─────────────────────────────────────")
		fmt.Println(output)
		fmt.Println("─────────────────────────────────────")
	}
}

// DisplayTrace displays a detailed ReAct execution trace
func DisplayTrace(agent *spec.AgentSpec, showFull bool) {
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  ReAct Execution Trace                                       ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("🎯 Goal: %s\n", agent.Goal)
	fmt.Printf("📊 Status: %s\n", agent.Metadata.Status)
	fmt.Printf("⏱️  Total Time: %dms\n", agent.Metadata.ExecutionTimeMs)
	fmt.Printf("💰 Total Cost: $%.4f\n\n", agent.Metadata.TotalCost)

	// Find ReAct nodes with traces
	for _, nodeResult := range agent.Metadata.NodeResults {
		if nodeResult.ReActTrace == nil {
			continue
		}

		trace := nodeResult.ReActTrace
		fmt.Printf("═══════════════════════════════════════════════════════════════\n")
		fmt.Printf("Node: %s\n", nodeResult.NodeID)
		fmt.Printf("Iterations: %d | Time: %dms | Cost: $%.4f\n",
			trace.Iterations, trace.TotalThinkingTimeMs, trace.IterationsCost)
		fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

		for _, step := range trace.ThinkingSteps {
			fmt.Printf("┌─ Iteration %d ─────────────────────────────────────────────┐\n", step.Iteration)
			fmt.Printf("│ Duration: %dms | Cost: $%.4f\n", step.DurationMs, step.Cost)
			fmt.Printf("└──────────────────────────────────────────────────────────────┘\n\n")

			// Show thought
			thought := step.Thought
			if !showFull && len(thought) > 500 {
				thought = thought[:500] + "\n... [truncated, use --full to see all]"
			}

			fmt.Printf("💭 Thought:\n")
			fmt.Printf("   %s\n\n", strings.ReplaceAll(thought, "\n", "\n   "))

			// Show tool calls
			if len(step.ToolCalls) > 0 {
				for _, toolCall := range step.ToolCalls {
					fmt.Printf("🔧 Tool Call: %s\n", toolCall.ToolName)

					// Show arguments
					if len(toolCall.Arguments) > 0 {
						fmt.Printf("   Arguments:\n")
						for key, val := range toolCall.Arguments {
							fmt.Printf("     • %s: %v\n", key, val)
						}
					}

					// Show result or error
					fmt.Printf("   Duration: %dms\n", toolCall.DurationMs)

					if toolCall.Error != "" {
						fmt.Printf("   ❌ Error: %s\n", toolCall.Error)
					} else {
						resultStr := fmt.Sprintf("%v", toolCall.Result)
						if !showFull && len(resultStr) > 300 {
							resultStr = resultStr[:300] + "... [truncated]"
						}
						fmt.Printf("   ✅ Result:\n")
						fmt.Printf("      %s\n", strings.ReplaceAll(resultStr, "\n", "\n      "))
					}
					fmt.Println()
				}
			}

			fmt.Println()
		}

		// Show final output
		if nodeResult.Output != nil {
			fmt.Printf("═══════════════════════════════════════════════════════════════\n")
			fmt.Printf("🎬 Final Output:\n")
			fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

			outputStr := fmt.Sprintf("%v", nodeResult.Output)
			fmt.Printf("%s\n\n", outputStr)
		}
	}
}

// PrintLiveTraceHeader prints the header for live trace mode
func PrintLiveTraceHeader() {
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  🔍 ReAct Execution with Live Trace                         ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")
}

// PrintLiveTraceSummary prints the final summary for live trace mode
func PrintLiveTraceSummary(metadata *spec.Metadata, output string) {
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  ✨ Execution Complete                                        ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("⏱️  Total Time: %dms\n", metadata.ExecutionTimeMs)
	fmt.Printf("💰 Total Cost: $%.4f\n\n", metadata.TotalCost)

	fmt.Printf("📄 Final Output:\n")
	fmt.Printf("─────────────────────────────────────────────────────────────\n")
	fmt.Printf("%s\n", output)
	fmt.Printf("─────────────────────────────────────────────────────────────\n\n")
}

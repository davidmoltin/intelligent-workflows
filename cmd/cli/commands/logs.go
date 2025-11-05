package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/cli"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	workflowFilter string
	statusFilter   string
	limit          int
	follow         bool
)

var logsCmd = &cobra.Command{
	Use:   "logs [execution-id]",
	Short: "View workflow execution logs",
	Long: `View execution logs for workflows. Can show a specific execution or list recent executions.

Examples:
  workflow logs                                    # List recent executions
  workflow logs abc123                             # Show specific execution
  workflow logs --workflow my-workflow             # Filter by workflow
  workflow logs --status completed                 # Filter by status
  workflow logs --limit 50                         # Show last 50 executions`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Create API client
		apiURL := viper.GetString("api.url")
		apiToken := viper.GetString("api.token")
		client := cli.NewClient(apiURL, apiToken)

		// Check API health
		if err := client.HealthCheck(); err != nil {
			fmt.Printf("❌ API health check failed: %v\n", err)
			fmt.Println("💡 Tip: Make sure the API server is running")
			os.Exit(1)
		}

		// If execution ID provided, show specific execution
		if len(args) == 1 {
			executionID := args[0]
			showExecutionDetails(client, executionID)
			return
		}

		// Otherwise, list executions
		listExecutions(client)
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().StringVar(&workflowFilter, "workflow", "", "Filter by workflow ID")
	logsCmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status (pending, running, completed, failed)")
	logsCmd.Flags().IntVar(&limit, "limit", 20, "Number of executions to show")
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow execution logs in real-time (requires execution ID)")
}

func showExecutionDetails(client *cli.Client, executionID string) {
	fmt.Printf("🔍 Fetching execution: %s\n\n", executionID)

	execution, err := client.GetExecution(executionID)
	if err != nil {
		fmt.Printf("❌ Failed to get execution: %v\n", err)
		os.Exit(1)
	}

	if outputJSON {
		data, _ := json.MarshalIndent(execution, "", "  ")
		fmt.Println(string(data))
		return
	}

	printExecutionDetails(execution)
}

func listExecutions(client *cli.Client) {
	fmt.Println("📋 Fetching executions...")

	executions, err := client.GetExecutions(workflowFilter)
	if err != nil {
		fmt.Printf("❌ Failed to get executions: %v\n", err)
		os.Exit(1)
	}

	// Apply filters
	if statusFilter != "" {
		filtered := executions[:0]
		for _, e := range executions {
			if string(e.Status) == statusFilter {
				filtered = append(filtered, e)
			}
		}
		executions = filtered
	}

	// Apply limit
	if len(executions) > limit {
		executions = executions[:limit]
	}

	if outputJSON {
		data, _ := json.MarshalIndent(executions, "", "  ")
		fmt.Println(string(data))
		return
	}

	printExecutionList(executions)
}

func printExecutionDetails(execution *models.WorkflowExecution) {
	fmt.Println("📊 Execution Details")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("ID:           %s\n", execution.ID)
	fmt.Printf("Workflow:     %s\n", execution.WorkflowID.String())
	fmt.Printf("Status:       %s\n", getStatusEmoji(execution.Status))
	fmt.Printf("Started:      %s\n", execution.StartedAt.Format("2006-01-02 15:04:05"))
	if execution.CompletedAt != nil {
		fmt.Printf("Completed:    %s\n", execution.CompletedAt.Format("2006-01-02 15:04:05"))
		duration := execution.CompletedAt.Sub(execution.StartedAt)
		fmt.Printf("Duration:     %s\n", duration.Round(time.Millisecond))
	}
	if execution.ErrorMessage != nil && *execution.ErrorMessage != "" {
		fmt.Printf("Error:        %s\n", *execution.ErrorMessage)
	}

	fmt.Println("\n📝 Execution Metadata:")
	fmt.Println("───────────────────────────────────────────────────────────")

	if execution.Metadata != nil {
		metadataData, err := json.MarshalIndent(execution.Metadata, "", "  ")
		if err == nil {
			fmt.Println(string(metadataData))
		}
	} else {
		fmt.Println("No metadata available")
	}

	fmt.Println("\n📦 Context:")
	fmt.Println("───────────────────────────────────────────────────────────")
	if execution.Context != nil {
		contextData, err := json.MarshalIndent(execution.Context, "", "  ")
		if err == nil {
			fmt.Println(string(contextData))
		}
	} else {
		fmt.Println("No context data available")
	}
}

func printExecutionList(executions []models.WorkflowExecution) {
	if len(executions) == 0 {
		fmt.Println("📭 No executions found")
		fmt.Println("\n💡 Test a workflow:")
		fmt.Println("  workflow test workflow.json --event event.json")
		return
	}

	fmt.Printf("📋 Found %d execution(s):\n\n", len(executions))
	fmt.Println("┌──────────────────────────────────────┬──────────────────────────┬────────────┬─────────────────────┐")
	fmt.Println("│ Execution ID                         │ Workflow                 │ Status     │ Started At          │")
	fmt.Println("├──────────────────────────────────────┼──────────────────────────┼────────────┼─────────────────────┤")

	for _, e := range executions {
		executionID := truncate(e.ID.String(), 36)
		workflowID := truncate(e.WorkflowID.String(), 24)
		status := getStatusEmoji(e.Status)
		startedAt := e.StartedAt.Format("2006-01-02 15:04:05")

		fmt.Printf("│ %-36s │ %-24s │ %-10s │ %-19s │\n", executionID, workflowID, status, startedAt)
	}

	fmt.Println("└──────────────────────────────────────┴──────────────────────────┴────────────┴─────────────────────┘")
	fmt.Println("\n📖 View details:")
	fmt.Println("  workflow logs <execution-id>")
}

func getStatusEmoji(status models.ExecutionStatus) string {
	switch status {
	case "pending":
		return "⏳ Pending"
	case "running":
		return "🏃 Running"
	case "completed":
		return "✅ Completed"
	case "failed":
		return "❌ Failed"
	case "blocked":
		return "🚫 Blocked"
	default:
		return string(status)
	}
}

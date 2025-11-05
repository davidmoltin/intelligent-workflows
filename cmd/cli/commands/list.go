package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/davidmoltin/intelligent-workflows/internal/cli"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	enabledOnly  bool
	disabledOnly bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workflows",
	Long: `List all workflows from the workflow engine server.

Examples:
  workflow list
  workflow list --enabled-only
  workflow list --json`,
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

		// Get workflows
		workflows, err := client.GetWorkflows()
		if err != nil {
			fmt.Printf("❌ Failed to get workflows: %v\n", err)
			os.Exit(1)
		}

		// Filter if needed
		if enabledOnly {
			filtered := workflows[:0]
			for _, w := range workflows {
				if w.Enabled {
					filtered = append(filtered, w)
				}
			}
			workflows = filtered
		} else if disabledOnly {
			filtered := workflows[:0]
			for _, w := range workflows {
				if !w.Enabled {
					filtered = append(filtered, w)
				}
			}
			workflows = filtered
		}

		// Output results
		if outputJSON {
			data, err := json.MarshalIndent(workflows, "", "  ")
			if err != nil {
				fmt.Printf("❌ Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(data))
		} else {
			printWorkflowList(workflows)
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&enabledOnly, "enabled-only", false, "Show only enabled workflows")
	listCmd.Flags().BoolVar(&disabledOnly, "disabled-only", false, "Show only disabled workflows")
}

func printWorkflowList(workflows []models.Workflow) {
	if len(workflows) == 0 {
		fmt.Println("📭 No workflows found")
		fmt.Println("\n💡 Create a new workflow:")
		fmt.Println("  workflow init my-workflow --template approval")
		return
	}

	fmt.Printf("\n📋 Found %d workflow(s):\n\n", len(workflows))
	fmt.Println("┌────────────────────────────────────────┬──────────────────────────┬─────────┬─────────┐")
	fmt.Println("│ Workflow ID                            │ Name                     │ Version │ Status  │")
	fmt.Println("├────────────────────────────────────────┼──────────────────────────┼─────────┼─────────┤")

	for _, w := range workflows {
		status := "✅ Enabled"
		if !w.Enabled {
			status = "❌ Disabled"
		}

		workflowID := truncate(w.WorkflowID, 38)
		name := truncate(w.Name, 24)
		version := truncate(w.Version, 7)

		fmt.Printf("│ %-38s │ %-24s │ %-7s │ %-7s │\n", workflowID, name, version, status)
	}

	fmt.Println("└────────────────────────────────────────┴──────────────────────────┴─────────┴─────────┘")
	fmt.Println("\n📖 View details:")
	fmt.Println("  workflow logs <execution-id>")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

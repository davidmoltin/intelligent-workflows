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
	updateIfExists bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy [workflow-file]",
	Short: "Deploy a workflow to the server",
	Long: `Deploy a workflow definition to the workflow engine server.

The deploy command will:
  1. Validate the workflow definition
  2. Check if the API server is reachable
  3. Create or update the workflow on the server
  4. Enable the workflow (if not disabled in definition)

Examples:
  workflow deploy workflow.json
  workflow deploy approval-workflow.json --update
  workflow deploy workflow.json --api-url http://prod.example.com:8080`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]

		// Check if file exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fmt.Printf("❌ Error: File '%s' not found\n", filename)
			os.Exit(1)
		}

		// Validate first
		fmt.Println("🔍 Validating workflow...")
		validationResult, err := cli.ValidateWorkflowFile(filename)
		if err != nil {
			fmt.Printf("❌ Error validating workflow: %v\n", err)
			os.Exit(1)
		}

		if !validationResult.Valid {
			fmt.Println("❌ Workflow validation failed:")
			for _, err := range validationResult.Errors {
				fmt.Printf("  - %s\n", err)
			}
			os.Exit(1)
		}
		fmt.Println("✅ Validation passed")

		// Load workflow
		workflow, err := cli.LoadWorkflowFromFile(filename)
		if err != nil {
			fmt.Printf("❌ Error loading workflow: %v\n", err)
			os.Exit(1)
		}

		// Create API client
		apiURL := viper.GetString("api.url")
		apiToken := viper.GetString("api.token")
		client := cli.NewClient(apiURL, apiToken)

		// Check API health
		fmt.Printf("🔗 Connecting to API: %s\n", apiURL)
		if err := client.HealthCheck(); err != nil {
			fmt.Printf("❌ API health check failed: %v\n", err)
			fmt.Println("💡 Tip: Make sure the API server is running")
			os.Exit(1)
		}

		// Check if workflow already exists
		existing, err := client.GetWorkflow(workflow.WorkflowID)
		if err == nil && existing != nil {
			// Workflow exists
			if !updateIfExists {
				fmt.Printf("❌ Workflow '%s' already exists\n", workflow.WorkflowID)
				fmt.Println("💡 Use --update flag to update the existing workflow")
				os.Exit(1)
			}

			// Update existing workflow
			fmt.Printf("🔄 Updating workflow '%s'...\n", workflow.Name)
			updated, err := client.UpdateWorkflow(workflow.WorkflowID, workflow)
			if err != nil {
				fmt.Printf("❌ Failed to update workflow: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Workflow updated successfully!")
			printWorkflowInfo(updated)
		} else {
			// Create new workflow
			fmt.Printf("🚀 Deploying workflow '%s'...\n", workflow.Name)
			created, err := client.CreateWorkflow(workflow)
			if err != nil {
				fmt.Printf("❌ Failed to deploy workflow: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Workflow deployed successfully!")
			printWorkflowInfo(created)
		}

		fmt.Println("\n📋 Next steps:")
		fmt.Printf("  • List workflows:   workflow list\n")
		fmt.Printf("  • Test workflow:    workflow test %s --event event.json\n", filename)
		fmt.Printf("  • View logs:        workflow logs <execution-id>\n")
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().BoolVarP(&updateIfExists, "update", "u", false, "Update workflow if it already exists")
}

func printWorkflowInfo(workflow *models.Workflow) {
	if outputJSON {
		data, _ := json.MarshalIndent(workflow, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("\n📦 Workflow Details:\n")
		fmt.Printf("  ID:         %s\n", workflow.ID)
		fmt.Printf("  Workflow:   %s\n", workflow.WorkflowID)
		fmt.Printf("  Name:       %s\n", workflow.Name)
		fmt.Printf("  Version:    %s\n", workflow.Version)
		fmt.Printf("  Enabled:    %v\n", workflow.Enabled)
		fmt.Printf("  Created:    %s\n", workflow.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated:    %s\n", workflow.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
}

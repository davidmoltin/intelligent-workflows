package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/davidmoltin/intelligent-workflows/internal/cli"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [workflow-file]",
	Short: "Validate a workflow definition",
	Long: `Validate a workflow definition file to ensure it meets all requirements.

The validator checks:
  - Required fields (workflow_id, name, version)
  - Valid trigger types
  - Step structure and configuration
  - Condition and action syntax
  - Reference consistency

Examples:
  workflow validate workflow.json
  workflow validate approval-workflow.json --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]

		// Check if file exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			fmt.Printf("❌ Error: File '%s' not found\n", filename)
			os.Exit(1)
		}

		// Validate the workflow
		result, err := cli.ValidateWorkflowFile(filename)
		if err != nil {
			fmt.Printf("❌ Error validating workflow: %v\n", err)
			os.Exit(1)
		}

		// Output results
		if outputJSON {
			outputValidationJSON(result)
		} else {
			outputValidationText(result, filename)
		}

		if !result.Valid {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func outputValidationText(result *cli.ValidationResult, filename string) {
	fmt.Printf("\n🔍 Validating workflow: %s\n\n", filename)

	if result.Valid {
		fmt.Println("✅ Workflow is valid!")
		fmt.Println("\nNext step:")
		fmt.Printf("  workflow deploy %s\n", filename)
	} else {
		fmt.Printf("❌ Workflow validation failed with %d error(s):\n\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
		fmt.Println("\n💡 Tip: Fix the errors above and run validate again")
	}
}

func outputValidationJSON(result *cli.ValidationResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

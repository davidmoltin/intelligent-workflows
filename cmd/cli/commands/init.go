package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/cli"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/spf13/cobra"
)

var (
	templateName string
	outputFile   string
)

var initCmd = &cobra.Command{
	Use:   "init [workflow-name]",
	Short: "Initialize a new workflow",
	Long: `Initialize a new workflow from a template or create a blank workflow.

Available templates:
  - approval: Order approval workflow for high-value transactions
  - fraud: Fraud detection and prevention workflow
  - inventory: Low inventory alert and replenishment workflow
  - payment: Payment failure handling workflow
  - customer: New customer onboarding workflow
  - blank: Empty workflow template

Examples:
  workflow init my-approval-workflow --template approval
  workflow init custom-workflow --template blank --output custom.json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflowName := args[0]

		// Determine output file name
		if outputFile == "" {
			outputFile = strings.ToLower(strings.ReplaceAll(workflowName, " ", "-")) + ".json"
		}

		// Check if file already exists
		if _, err := os.Stat(outputFile); err == nil {
			fmt.Printf("❌ Error: File '%s' already exists\n", outputFile)
			os.Exit(1)
		}

		// Load template
		workflow, err := loadTemplate(templateName, workflowName)
		if err != nil {
			fmt.Printf("❌ Error loading template: %v\n", err)
			os.Exit(1)
		}

		// Save to file
		if err := cli.SaveWorkflowToFile(workflow, outputFile); err != nil {
			fmt.Printf("❌ Error saving workflow: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Created workflow '%s' from template '%s'\n", workflowName, templateName)
		fmt.Printf("📄 File: %s\n", outputFile)
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Edit the workflow: %s\n", outputFile)
		fmt.Printf("  2. Validate: workflow validate %s\n", outputFile)
		fmt.Printf("  3. Deploy: workflow deploy %s\n", outputFile)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&templateName, "template", "t", "blank", "Template to use (approval, fraud, inventory, payment, customer, blank)")
	initCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file name (default: <workflow-name>.json)")
}

func loadTemplate(template, name string) (*models.Workflow, error) {
	// First try to load from templates directory
	templatePath := filepath.Join("templates", "workflows", template+".json")
	if _, err := os.Stat(templatePath); err == nil {
		workflow, err := cli.LoadWorkflowFromFile(templatePath)
		if err != nil {
			return nil, err
		}
		workflow.Name = name
		return workflow, nil
	}

	// Otherwise use built-in templates
	switch template {
	case "approval":
		return createApprovalTemplate(name), nil
	case "fraud":
		return createFraudTemplate(name), nil
	case "inventory":
		return createInventoryTemplate(name), nil
	case "payment":
		return createPaymentTemplate(name), nil
	case "customer":
		return createCustomerTemplate(name), nil
	case "blank":
		return createBlankTemplate(name), nil
	default:
		return nil, fmt.Errorf("unknown template: %s", template)
	}
}

func createBlankTemplate(name string) *models.Workflow {
	return &models.Workflow{
		WorkflowID: strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Version:    "1.0.0",
		Name:       name,
		Enabled:    true,
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type: "custom",
			},
			Steps: []models.Step{
				{
					ID:   "step-1",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}
}

func createApprovalTemplate(name string) *models.Workflow {
	return &models.Workflow{
		WorkflowID: strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Version:    "1.0.0",
		Name:       name,
		Enabled:    true,
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type: "order.created",
			},
			Context: models.ContextDefinition{
				Load: []string{"order.details", "customer.profile"},
			},
			Steps: []models.Step{
				{
					ID:   "check-order-value",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "order.total",
						Operator: "gt",
						Value:    1000,
					},
					OnTrue:  "require-approval",
					OnFalse: "auto-approve",
				},
				{
					ID:   "require-approval",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:       "notify",
							Recipients: []string{"manager@example.com", "finance@example.com"},
							Message:    "Order approval required for high value transaction",
						},
					},
					Metadata: map[string]interface{}{
						"approval_required": true,
						"next_on_approve":   "process-order",
						"next_on_reject":    "cancel-order",
					},
				},
				{
					ID:   "auto-approve",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
				{
					ID:   "process-order",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
				{
					ID:   "cancel-order",
					Type: "action",
					Action: &models.Action{
						Type: "block",
					},
				},
			},
		},
	}
}

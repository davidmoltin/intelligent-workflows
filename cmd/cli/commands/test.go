package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/cli"
	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	eventFile string
	waitForResult bool
	timeout int
)

var testCmd = &cobra.Command{
	Use:   "test [workflow-file]",
	Short: "Test a workflow with sample data",
	Long: `Test a workflow by sending a test event to the workflow engine.

The test command will:
  1. Load the workflow definition
  2. Load the event data from file or use a sample event
  3. Send the event to the API
  4. Optionally wait for execution results

Examples:
  workflow test workflow.json --event test-event.json
  workflow test workflow.json --event order.json --wait
  workflow test workflow.json (uses sample event)`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflowFile := args[0]

		// Check if workflow file exists
		if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
			fmt.Printf("❌ Error: Workflow file '%s' not found\n", workflowFile)
			os.Exit(1)
		}

		// Load workflow
		workflow, err := cli.LoadWorkflowFromFile(workflowFile)
		if err != nil {
			fmt.Printf("❌ Error loading workflow: %v\n", err)
			os.Exit(1)
		}

		// Load or create event
		var event *models.Event
		if eventFile != "" {
			// Load event from file
			eventData, err := os.ReadFile(eventFile)
			if err != nil {
				fmt.Printf("❌ Error reading event file: %v\n", err)
				os.Exit(1)
			}

			event = &models.Event{}
			if err := json.Unmarshal(eventData, event); err != nil {
				fmt.Printf("❌ Error parsing event file: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Create sample event based on trigger type
			event = createSampleEvent(workflow.Definition.Trigger.Type)
		}

		// Ensure event has required fields
		if event.ID == uuid.Nil {
			event.ID = uuid.New()
		}
		if event.EventType == "" {
			event.EventType = workflow.Definition.Trigger.Type
		}
		if event.ReceivedAt.IsZero() {
			event.ReceivedAt = time.Now()
		}

		// Create API client
		apiURL := viper.GetString("api.url")
		apiToken := viper.GetString("api.token")
		client := cli.NewClient(apiURL, apiToken)

		// Check API health
		fmt.Println("🔗 Connecting to API...")
		if err := client.HealthCheck(); err != nil {
			fmt.Printf("❌ API health check failed: %v\n", err)
			os.Exit(1)
		}

		// Send event
		fmt.Printf("🚀 Sending test event (type: %s)...\n", event.EventType)
		if err := client.CreateEvent(event); err != nil {
			fmt.Printf("❌ Failed to send event: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Event sent successfully!")
		fmt.Printf("\n📋 Event Details:\n")
		fmt.Printf("  ID:        %s\n", event.ID)
		fmt.Printf("  Type:      %s\n", event.EventType)
		fmt.Printf("  Timestamp: %s\n", event.ReceivedAt.Format("2006-01-02 15:04:05"))

		if waitForResult {
			fmt.Printf("\n⏳ Waiting for execution results (timeout: %ds)...\n", timeout)
			// Wait and poll for results
			// Note: This requires the execution API to be implemented
			// For now, just show the message
			fmt.Println("💡 Use 'workflow logs' to view execution results")
		} else {
			fmt.Println("\n💡 Next steps:")
			fmt.Println("  • View executions: workflow logs")
		}
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVarP(&eventFile, "event", "e", "", "Event data file (JSON)")
	testCmd.Flags().BoolVarP(&waitForResult, "wait", "w", false, "Wait for execution results")
	testCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "Timeout in seconds when waiting")
}

func createSampleEvent(triggerType string) *models.Event {
	event := &models.Event{
		ID:         uuid.New(),
		EventType:  triggerType,
		ReceivedAt: time.Now(),
		Source:     "cli-test",
	}

	// Create sample payload based on trigger type
	switch triggerType {
	case "order.created":
		event.Payload = models.JSONB{
			"order": map[string]interface{}{
				"id":          uuid.New().String(),
				"customer_id": "cust-123",
				"total":       1500.00,
				"currency":    "USD",
				"status":      "pending",
				"items": []map[string]interface{}{
					{
						"product_id": "prod-456",
						"quantity":   2,
						"price":      750.00,
					},
				},
				"shipping_address": map[string]interface{}{
					"country": "US",
					"city":    "New York",
					"zip":     "10001",
				},
				"payment_method": "credit_card",
			},
		}
	case "payment.failed":
		event.Payload = models.JSONB{
			"order": map[string]interface{}{
				"id": uuid.New().String(),
				"customer": map[string]interface{}{
					"email": "customer@example.com",
				},
			},
			"payment": map[string]interface{}{
				"failure_reason": "insufficient_funds",
				"amount":         99.99,
			},
		}
	case "inventory.low":
		event.Payload = models.JSONB{
			"product": map[string]interface{}{
				"id":            "prod-789",
				"name":          "Sample Product",
				"quantity":      5,
				"reorder_point": 10,
			},
		}
	case "customer.created":
		event.Payload = models.JSONB{
			"customer": map[string]interface{}{
				"id":    uuid.New().String(),
				"email": "newcustomer@example.com",
				"name":  "John Doe",
			},
		}
	default:
		event.Payload = models.JSONB{
			"data": map[string]interface{}{
				"test": true,
			},
		}
	}

	return event
}

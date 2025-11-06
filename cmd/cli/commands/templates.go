package commands

import (
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
)

func createFraudTemplate(name string) *models.Workflow {
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
				Load: []string{"order.details", "customer.history"},
			},
			Steps: []models.Step{
				{
					ID:   "check-high-risk",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "order.total",
						Operator: "gt",
						Value:    5000,
					},
					OnTrue:  "fraud-review",
					OnFalse: "check-velocity",
				},
				{
					ID:   "check-velocity",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "customer.order_count_24h",
						Operator: "gt",
						Value:    5,
					},
					OnTrue:  "fraud-review",
					OnFalse: "allow-order",
				},
				{
					ID:   "fraud-review",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:       "notify",
							Recipients: []string{"fraud-team@example.com"},
							Message:    "High-risk order detected - manual review required",
						},
					},
				},
				{
					ID:   "allow-order",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
				{
					ID:   "block-order",
					Type: "action",
					Action: &models.Action{
						Type: "block",
					},
				},
			},
		},
	}
}

func createInventoryTemplate(name string) *models.Workflow {
	return &models.Workflow{
		WorkflowID: strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Version:    "1.0.0",
		Name:       name,
		Enabled:    true,
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type: "inventory.low",
			},
			Context: models.ContextDefinition{
				Load: []string{"product.details", "inventory.levels"},
			},
			Steps: []models.Step{
				{
					ID:   "check-reorder-needed",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "product.quantity",
						Operator: "lt",
						Value:    "product.reorder_point",
					},
					OnTrue:  "notify-team",
					OnFalse: "no-action",
				},
				{
					ID:   "notify-team",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:   "webhook",
							URL:    "https://api.example.com/notify",
							Method: "POST",
							Body: map[string]interface{}{
								"product_id": "{{.context.product_id}}",
								"quantity":   "{{.context.current_quantity}}",
								"message":    "Low inventory alert - reorder needed",
							},
						},
					},
				},
				{
					ID:   "create-purchase-order",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:   "create_record",
							Entity: "purchase_order",
							Data: map[string]interface{}{
								"product_id": "{{.context.product_id}}",
								"quantity":   "{{.context.reorder_quantity}}",
							},
						},
					},
				},
				{
					ID:   "no-action",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}
}

func createPaymentTemplate(name string) *models.Workflow {
	return &models.Workflow{
		WorkflowID: strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Version:    "1.0.0",
		Name:       name,
		Enabled:    true,
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type: "payment.failed",
			},
			Context: models.ContextDefinition{
				Load: []string{"order.details", "customer.email"},
			},
			Steps: []models.Step{
				{
					ID:   "notify-customer",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:   "webhook",
							URL:    "https://api.example.com/email/send",
							Method: "POST",
							Body: map[string]interface{}{
								"to":       "{{.context.customer_email}}",
								"subject":  "Payment Failed - Action Required",
								"template": "payment-failed",
							},
						},
					},
				},
				{
					ID:   "wait-for-retry",
					Type: "wait",
					Wait: &models.WaitConfig{
						Event:     "payment.retry",
						Timeout:   "24h",
						OnTimeout: "cancel-order",
					},
				},
				{
					ID:   "check-payment-status",
					Type: "condition",
					Condition: &models.Condition{
						Field:    "payment.status",
						Operator: "eq",
						Value:    "pending",
					},
					OnTrue:  "cancel-order",
					OnFalse: "complete",
				},
				{
					ID:   "cancel-order",
					Type: "action",
					Action: &models.Action{
						Type: "block",
					},
				},
				{
					ID:   "complete",
					Type: "action",
					Action: &models.Action{
						Type: "allow",
					},
				},
			},
		},
	}
}

func createCustomerTemplate(name string) *models.Workflow {
	return &models.Workflow{
		WorkflowID: strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Version:    "1.0.0",
		Name:       name,
		Enabled:    true,
		Definition: models.WorkflowDefinition{
			Trigger: models.TriggerDefinition{
				Type: "customer.created",
			},
			Context: models.ContextDefinition{
				Load: []string{"customer.details"},
			},
			Steps: []models.Step{
				{
					ID:   "send-welcome-email",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:   "webhook",
							URL:    "https://api.example.com/email/send",
							Method: "POST",
							Body: map[string]interface{}{
								"to":       "{{.context.customer_email}}",
								"subject":  "Welcome to Our Store!",
								"template": "customer-welcome",
							},
						},
					},
				},
				{
					ID:   "add-to-loyalty-program",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:   "create_record",
							Entity: "loyalty_member",
							Data: map[string]interface{}{
								"customer_id": "{{.context.customer_id}}",
								"tier":        "bronze",
							},
						},
					},
				},
				{
					ID:   "schedule-followup",
					Type: "wait",
					Wait: &models.WaitConfig{
						Event:     "customer.followup",
						Timeout:   "168h", // 7 days
						OnTimeout: "send-survey",
					},
				},
				{
					ID:   "send-survey",
					Type: "execute",
					Execute: []models.ExecuteAction{
						{
							Type:   "webhook",
							URL:    "https://api.example.com/email/send",
							Method: "POST",
							Body: map[string]interface{}{
								"to":       "{{.context.customer_email}}",
								"subject":  "How are we doing?",
								"template": "customer-survey",
							},
						},
					},
				},
			},
		},
	}
}

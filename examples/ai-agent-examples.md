# AI Agent Integration Examples

This document demonstrates how AI agents can interact with the Intelligent Workflows service.

## 1. Creating Workflows via Natural Language

### Example 1: Simple Order Workflow

**AI Agent Request:**
```json
POST /api/v1/ai/interpret
{
  "prompt": "Block all orders over $5000 from customers who signed up in the last 30 days",
  "context": {
    "agent_id": "claude-agent-001",
    "session_id": "session-abc123"
  }
}
```

**Service Response:**
```json
{
  "interpretation": {
    "action": "create_workflow",
    "confidence": 0.96,
    "entities_identified": ["order", "customer"],
    "conditions": [
      {
        "field": "order.total",
        "operator": "gt",
        "value": 5000,
        "type": "number"
      },
      {
        "field": "customer.created_at",
        "operator": "gt",
        "value": "now-30d",
        "type": "date"
      }
    ],
    "action_type": "block",
    "trigger_event": "order.checkout.initiated"
  },
  "suggested_workflow": {
    "workflow_id": "new_customer_high_value_block",
    "name": "Block High Value Orders from New Customers",
    "trigger": {
      "type": "event",
      "event": "order.checkout.initiated"
    },
    "steps": [
      {
        "id": "check_conditions",
        "type": "condition",
        "condition": {
          "and": [
            {
              "field": "order.total",
              "operator": "gt",
              "value": 5000
            },
            {
              "field": "customer.account_age_days",
              "operator": "lte",
              "value": 30
            }
          ]
        },
        "on_true": "block_order",
        "on_false": "allow_order"
      },
      {
        "id": "block_order",
        "type": "action",
        "action": "block",
        "reason": "High value order from new customer requires review"
      },
      {
        "id": "allow_order",
        "type": "action",
        "action": "allow"
      }
    ]
  },
  "requires_confirmation": true,
  "explanation": "I've created a workflow that will:\n1. Trigger when a customer initiates checkout\n2. Check if the order total exceeds $5000\n3. Check if the customer account is less than 30 days old\n4. Block the order if both conditions are true\n5. Otherwise, allow the order to proceed"
}
```

**AI Agent Confirms and Deploys:**
```json
POST /api/v1/workflows
{
  "workflow_id": "new_customer_high_value_block",
  "version": "1.0.0",
  "name": "Block High Value Orders from New Customers",
  "enabled": true,
  ...
}
```

---

## 2. Querying System Capabilities

**AI Agent Request:**
```json
GET /api/v1/ai/capabilities
```

**Service Response:**
```json
{
  "version": "1.0.0",
  "entities": {
    "order": {
      "description": "Customer order entity",
      "fields": {
        "id": {"type": "string", "description": "Unique order identifier"},
        "total": {"type": "number", "description": "Order total amount"},
        "status": {"type": "string", "enum": ["draft", "pending", "confirmed", "shipped", "delivered", "cancelled"]},
        "customer_id": {"type": "string", "description": "Associated customer ID"},
        "items": {"type": "array", "description": "Order line items"},
        "created_at": {"type": "datetime", "description": "Order creation timestamp"},
        "shipping_address": {"type": "object", "description": "Shipping address details"}
      },
      "events": [
        "order.created",
        "order.updated",
        "order.checkout.initiated",
        "order.payment.received",
        "order.shipped",
        "order.delivered",
        "order.cancelled"
      ],
      "available_actions": ["allow", "block", "require_approval"]
    },
    "cart": {
      "description": "Shopping cart entity",
      "fields": {
        "id": {"type": "string"},
        "customer_id": {"type": "string"},
        "items": {"type": "array"},
        "total": {"type": "number"},
        "created_at": {"type": "datetime"},
        "updated_at": {"type": "datetime"}
      },
      "events": [
        "cart.created",
        "cart.item.added",
        "cart.item.removed",
        "cart.item.updated",
        "cart.checkout.started",
        "cart.abandoned"
      ],
      "available_actions": ["allow", "block", "notify_customer"]
    },
    "product": {
      "description": "Product catalog entity",
      "fields": {
        "id": {"type": "string"},
        "sku": {"type": "string"},
        "name": {"type": "string"},
        "price": {"type": "number"},
        "inventory_quantity": {"type": "number"},
        "available_for_sale": {"type": "boolean"},
        "category": {"type": "string"}
      },
      "events": [
        "product.created",
        "product.updated",
        "product.price.changed",
        "product.inventory.changed",
        "product.inventory.low",
        "product.out_of_stock"
      ],
      "available_actions": ["update_inventory", "update_price", "notify_subscribers"]
    },
    "quote": {
      "description": "Sales quote entity",
      "fields": {
        "id": {"type": "string"},
        "number": {"type": "string"},
        "customer_id": {"type": "string"},
        "total": {"type": "number"},
        "status": {"type": "string", "enum": ["draft", "sent", "accepted", "rejected", "expired"]},
        "expires_at": {"type": "datetime"},
        "created_at": {"type": "datetime"}
      },
      "events": [
        "quote.created",
        "quote.sent",
        "quote.viewed",
        "quote.accepted",
        "quote.rejected",
        "quote.expired",
        "quote.expiring_soon"
      ],
      "available_actions": ["allow", "require_approval", "extend_expiration", "convert_to_order"]
    },
    "customer": {
      "description": "Customer entity",
      "fields": {
        "id": {"type": "string"},
        "email": {"type": "string"},
        "name": {"type": "string"},
        "tier": {"type": "string", "enum": ["standard", "silver", "gold", "platinum"]},
        "credit_limit": {"type": "number"},
        "account_age_days": {"type": "number"},
        "total_orders": {"type": "number"},
        "lifetime_value": {"type": "number"}
      },
      "events": [
        "customer.created",
        "customer.updated",
        "customer.tier.changed",
        "customer.credit_limit.changed"
      ],
      "available_actions": ["update_tier", "update_credit_limit", "flag_account"]
    }
  },
  "operators": {
    "comparison": ["eq", "neq", "gt", "gte", "lt", "lte"],
    "string": ["contains", "starts_with", "ends_with", "matches"],
    "array": ["in", "not_in", "contains", "has_all", "has_any"],
    "logical": ["and", "or", "not"],
    "date": ["before", "after", "between", "within_last", "older_than"]
  },
  "action_types": {
    "allow": {
      "description": "Allow the operation to proceed",
      "parameters": []
    },
    "block": {
      "description": "Prevent the operation from proceeding",
      "parameters": ["reason", "metadata"]
    },
    "require_approval": {
      "description": "Block and create approval request",
      "parameters": ["approver_role", "expires_in", "priority"]
    }
  },
  "execution_types": {
    "notify": {
      "description": "Send notification",
      "channels": ["email", "slack", "sms", "webhook", "pagerduty"],
      "parameters": ["recipients", "message", "template"]
    },
    "webhook": {
      "description": "Call external webhook",
      "parameters": ["url", "method", "headers", "body"]
    },
    "http_request": {
      "description": "Make HTTP request to external API",
      "parameters": ["url", "method", "headers", "body", "timeout"]
    },
    "create_record": {
      "description": "Create new record in system",
      "parameters": ["entity", "fields"]
    },
    "update_record": {
      "description": "Update existing record",
      "parameters": ["entity", "entity_id", "fields"]
    },
    "create_approval_request": {
      "description": "Create approval request",
      "parameters": ["entity", "entity_id", "approver_role", "expires_in"]
    }
  },
  "context_sources": [
    "order.details",
    "order.history",
    "customer.profile",
    "customer.order_history",
    "customer.credit_limit",
    "product.details",
    "product.inventory",
    "cart.details",
    "quote.details"
  ],
  "trigger_types": ["event", "schedule", "manual", "webhook"]
}
```

---

## 3. Complex Multi-Step Workflow Creation

**AI Agent Request:**
```json
POST /api/v1/ai/interpret
{
  "prompt": "I need a workflow that:\n1. Triggers when inventory goes below 10 units\n2. Checks if the product is marked as critical\n3. If critical, immediately notify the warehouse team via Slack and email\n4. Create a purchase order for 50 units\n5. If not critical, just log it for the weekly review",
  "context": {
    "agent_id": "claude-agent-001",
    "mode": "create"
  }
}
```

**Service Response:**
```json
{
  "interpretation": {
    "action": "create_workflow",
    "confidence": 0.93,
    "workflow_type": "conditional_with_parallel_actions",
    "trigger": {
      "type": "event",
      "event": "product.inventory.low"
    },
    "steps_identified": 5,
    "requires_external_integration": true,
    "external_systems": ["slack", "email", "purchase_order_system"]
  },
  "suggested_workflow": {
    "workflow_id": "critical_inventory_management",
    "version": "1.0.0",
    "name": "Critical Inventory Management",
    "description": "Automated inventory replenishment for critical products",
    "trigger": {
      "type": "event",
      "event": "product.inventory.changed"
    },
    "context": {
      "load": ["product.details", "product.inventory", "product.suppliers"]
    },
    "steps": [
      {
        "id": "check_low_inventory",
        "type": "condition",
        "condition": {
          "field": "product.inventory_quantity",
          "operator": "lt",
          "value": 10
        },
        "on_true": "check_critical_status",
        "on_false": "skip_workflow"
      },
      {
        "id": "check_critical_status",
        "type": "condition",
        "condition": {
          "field": "product.is_critical",
          "operator": "eq",
          "value": true
        },
        "on_true": "handle_critical_inventory",
        "on_false": "log_for_review"
      },
      {
        "id": "handle_critical_inventory",
        "type": "action",
        "action": "execute",
        "execute": [
          {
            "type": "notify",
            "channel": "slack",
            "recipients": ["#warehouse-team"],
            "message": "🚨 CRITICAL: Product {{product.sku}} - {{product.name}} is low on inventory ({{product.inventory_quantity}} units remaining)"
          },
          {
            "type": "notify",
            "channel": "email",
            "recipients": ["role:warehouse_manager"],
            "template": "critical_inventory_alert",
            "data": {
              "product_name": "{{product.name}}",
              "product_sku": "{{product.sku}}",
              "current_quantity": "{{product.inventory_quantity}}",
              "reorder_quantity": 50
            }
          },
          {
            "type": "create_record",
            "entity": "purchase_order",
            "fields": {
              "product_id": "{{product.id}}",
              "quantity": 50,
              "supplier_id": "{{product.preferred_supplier_id}}",
              "priority": "urgent",
              "status": "draft",
              "created_by": "workflow_automation"
            }
          }
        ],
        "next": "complete"
      },
      {
        "id": "log_for_review",
        "type": "action",
        "action": "execute",
        "execute": [
          {
            "type": "create_record",
            "entity": "inventory_review_item",
            "fields": {
              "product_id": "{{product.id}}",
              "current_quantity": "{{product.inventory_quantity}}",
              "review_status": "pending",
              "created_at": "{{now}}"
            }
          }
        ],
        "next": "complete"
      },
      {
        "id": "skip_workflow",
        "type": "action",
        "action": "allow"
      },
      {
        "id": "complete",
        "type": "action",
        "action": "allow"
      }
    ]
  },
  "explanation": "I've created a comprehensive workflow that handles low inventory situations differently based on whether the product is marked as critical. For critical products, it sends immediate notifications and creates a purchase order. For non-critical products, it logs them for weekly review.",
  "warnings": [
    "This workflow requires integration with your purchase order system",
    "Ensure the 'is_critical' flag is properly set on products",
    "Verify Slack channel '#warehouse-team' exists"
  ],
  "suggested_tests": [
    {
      "scenario": "Critical product drops below threshold",
      "test_payload": {
        "product": {
          "id": "prod_123",
          "sku": "WIDGET-001",
          "name": "Critical Widget",
          "inventory_quantity": 8,
          "is_critical": true
        }
      },
      "expected_outcome": "Notifications sent, PO created"
    }
  ]
}
```

---

## 4. Workflow Validation

**AI Agent Request:**
```json
POST /api/v1/ai/validate
{
  "workflow": {
    "workflow_id": "test_workflow",
    "steps": [
      {
        "id": "step1",
        "type": "condition",
        "condition": {
          "field": "order.amount",
          "operator": "greater_than",
          "value": 1000
        }
      }
    ]
  }
}
```

**Service Response:**
```json
{
  "valid": false,
  "errors": [
    {
      "step_id": "step1",
      "field": "condition.operator",
      "message": "Invalid operator 'greater_than'. Did you mean 'gt' or 'gte'?",
      "severity": "error",
      "suggestion": "gt"
    },
    {
      "step_id": "step1",
      "field": "on_true",
      "message": "Missing required field 'on_true' for condition step",
      "severity": "error"
    },
    {
      "step_id": "step1",
      "field": "on_false",
      "message": "Missing required field 'on_false' for condition step",
      "severity": "error"
    }
  ],
  "warnings": [
    {
      "field": "trigger",
      "message": "No trigger specified. Workflow can only be executed manually.",
      "severity": "warning"
    }
  ],
  "suggestions": [
    "Add workflow description for better documentation",
    "Consider adding error handling for condition evaluation failures"
  ],
  "corrected_workflow": {
    "workflow_id": "test_workflow",
    "steps": [
      {
        "id": "step1",
        "type": "condition",
        "condition": {
          "field": "order.amount",
          "operator": "gt",
          "value": 1000
        },
        "on_true": "next_step",
        "on_false": "end"
      }
    ]
  }
}
```

---

## 5. Real-Time Workflow Execution Monitoring

**AI Agent Subscribes to Execution Events:**
```json
POST /api/v1/subscriptions
{
  "agent_id": "claude-agent-001",
  "events": ["execution.started", "execution.completed", "execution.failed"],
  "filters": {
    "workflow_id": ["high_value_order_approval"]
  }
}
```

**Service Streams Events via WebSocket:**
```json
{
  "event": "execution.started",
  "execution_id": "exec_abc123",
  "workflow_id": "high_value_order_approval",
  "trigger_event": "order.checkout.initiated",
  "timestamp": "2025-11-05T10:30:00Z",
  "context": {
    "order_id": "ord_456",
    "customer_id": "cust_789"
  }
}

{
  "event": "step.completed",
  "execution_id": "exec_abc123",
  "step_id": "check_order_value",
  "result": true,
  "next_step": "require_approval",
  "timestamp": "2025-11-05T10:30:01Z"
}

{
  "event": "execution.blocked",
  "execution_id": "exec_abc123",
  "step_id": "require_approval",
  "result": "blocked",
  "reason": "Order requires approval for amounts over $10,000",
  "approval_request_id": "appr_xyz",
  "timestamp": "2025-11-05T10:30:02Z"
}
```

---

## 6. AI Agent Taking Action on Approvals

**AI Agent Queries Pending Approvals:**
```json
GET /api/v1/approvals?status=pending&approver_role=ai_agent
```

**Service Response:**
```json
{
  "approvals": [
    {
      "id": "appr_xyz",
      "request_id": "req_123",
      "entity_type": "order",
      "entity_id": "ord_456",
      "execution_id": "exec_abc123",
      "reason": "Order requires approval for amounts over $10,000",
      "requested_at": "2025-11-05T10:30:02Z",
      "expires_at": "2025-11-06T10:30:02Z",
      "context": {
        "order": {
          "id": "ord_456",
          "total": 15000,
          "customer_id": "cust_789",
          "items_count": 3
        },
        "customer": {
          "id": "cust_789",
          "name": "Acme Corp",
          "tier": "gold",
          "lifetime_value": 250000,
          "payment_terms": "net_30"
        }
      }
    }
  ]
}
```

**AI Agent Analyzes and Approves:**
```json
POST /api/v1/approvals/appr_xyz/approve
{
  "approver_id": "claude-agent-001",
  "approver_type": "ai_agent",
  "reason": "Customer is Gold tier with $250K lifetime value and excellent payment history. Order within expected range.",
  "metadata": {
    "analysis": {
      "risk_score": 0.15,
      "confidence": 0.92,
      "factors_considered": [
        "customer_tier",
        "lifetime_value",
        "payment_history",
        "order_size_vs_historical_average"
      ]
    }
  }
}
```

---

## 7. Workflow Suggestions Based on Historical Data

**AI Agent Request:**
```json
POST /api/v1/ai/suggest
{
  "scenario": "We're seeing a lot of cart abandonment at checkout. What workflow could help?",
  "context": {
    "recent_events": ["cart.abandoned", "cart.checkout.started"],
    "business_goal": "reduce_cart_abandonment"
  }
}
```

**Service Response:**
```json
{
  "suggestions": [
    {
      "workflow_id": "cart_abandonment_recovery",
      "name": "Cart Abandonment Recovery",
      "description": "Automatically send recovery emails and offers when carts are abandoned",
      "relevance_score": 0.95,
      "estimated_impact": "15-25% recovery rate based on industry benchmarks",
      "workflow": {
        "trigger": {
          "type": "event",
          "event": "cart.abandoned"
        },
        "steps": [
          {
            "id": "wait_30_minutes",
            "type": "wait",
            "duration": "30m"
          },
          {
            "id": "check_still_abandoned",
            "type": "condition",
            "condition": {
              "field": "cart.status",
              "operator": "eq",
              "value": "abandoned"
            },
            "on_true": "send_recovery_email",
            "on_false": "end"
          },
          {
            "id": "send_recovery_email",
            "type": "action",
            "action": "execute",
            "execute": [
              {
                "type": "notify",
                "channel": "email",
                "recipients": ["{{cart.customer_email}}"],
                "template": "cart_recovery",
                "data": {
                  "cart_items": "{{cart.items}}",
                  "cart_total": "{{cart.total}}",
                  "discount_code": "{{generate_discount_code('COMEBACK10', 10)}}"
                }
              }
            ]
          }
        ]
      },
      "implementation_notes": [
        "Requires email template 'cart_recovery'",
        "Consider adding A/B testing for different discount amounts",
        "Set up tracking to measure conversion rate"
      ]
    },
    {
      "workflow_id": "checkout_assistance",
      "name": "Proactive Checkout Assistance",
      "description": "Offer help when customers are stuck at checkout",
      "relevance_score": 0.82,
      "estimated_impact": "10-15% reduction in abandonment",
      "workflow": {
        "trigger": {
          "type": "event",
          "event": "cart.checkout.started"
        },
        "steps": [
          {
            "id": "wait_for_delay",
            "type": "wait",
            "duration": "2m"
          },
          {
            "id": "check_still_in_checkout",
            "type": "condition",
            "condition": {
              "and": [
                {
                  "field": "cart.checkout_step",
                  "operator": "neq",
                  "value": "completed"
                },
                {
                  "field": "cart.checkout_duration_seconds",
                  "operator": "gt",
                  "value": 120
                }
              ]
            },
            "on_true": "offer_assistance",
            "on_false": "end"
          },
          {
            "id": "offer_assistance",
            "type": "action",
            "action": "execute",
            "execute": [
              {
                "type": "show_chat_widget",
                "message": "Need help completing your order? Our team is here to assist!"
              }
            ]
          }
        ]
      }
    }
  ],
  "data_insights": {
    "abandonment_rate": "68%",
    "common_abandonment_stages": ["payment_info", "shipping_address"],
    "peak_abandonment_times": ["weekday_evenings", "lunch_hours"]
  }
}
```

---

## 8. Authentication Flow for AI Agents

```http
POST /api/v1/auth/agent/token
Content-Type: application/json

{
  "agent_id": "claude-agent-001",
  "agent_type": "llm",
  "provider": "anthropic",
  "credentials": {
    "api_key": "sk-ant-...",
    "signature": "..."
  },
  "requested_permissions": [
    "workflows:read",
    "workflows:create",
    "workflows:execute",
    "executions:read",
    "approvals:read",
    "approvals:approve",
    "approvals:reject"
  ]
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "...",
  "permissions": [
    "workflows:read",
    "workflows:create",
    "workflows:execute",
    "executions:read",
    "approvals:read",
    "approvals:approve",
    "approvals:reject"
  ],
  "rate_limits": {
    "requests_per_minute": 60,
    "requests_per_hour": 1000,
    "concurrent_executions": 10
  }
}
```

---

## Summary

These examples demonstrate:
1. **Natural Language Processing**: AI agents can describe workflows in plain English
2. **Capability Discovery**: Agents can query what's possible in the system
3. **Complex Workflow Creation**: Multi-step workflows with conditional logic
4. **Validation**: Real-time validation with helpful error messages
5. **Monitoring**: Real-time execution tracking via WebSockets
6. **Decision Making**: AI agents can approve/reject requests based on context
7. **Suggestions**: AI can analyze patterns and suggest helpful workflows
8. **Authentication**: Secure, permission-based access for AI agents

The system is designed to be both human-friendly and AI-friendly, enabling seamless collaboration between human operators and AI agents.

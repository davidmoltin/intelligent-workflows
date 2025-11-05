# Workflow CLI Tool

The Intelligent Workflows CLI is a command-line tool for managing e-commerce workflows. It provides commands to create, validate, deploy, test, and monitor workflows.

## Installation

### Build from Source

```bash
go build -o workflow ./cmd/cli/
```

### Install to System

```bash
go install ./cmd/cli/
```

## Configuration

The CLI can be configured using:

1. **Configuration File**: `~/.workflow-cli.yaml`
2. **Environment Variables**: Prefix with `WORKFLOW_`
3. **Command-line Flags**: Override file and environment settings

### Configuration File Example

Create `~/.workflow-cli.yaml`:

```yaml
api:
  url: "http://localhost:8080"
  token: "your-api-token"
```

### Environment Variables

```bash
export WORKFLOW_API_URL="http://localhost:8080"
export WORKFLOW_API_TOKEN="your-api-token"
```

## Commands

### `workflow init` - Initialize a New Workflow

Create a new workflow from a template or start with a blank template.

**Usage:**
```bash
workflow init [workflow-name] [flags]
```

**Flags:**
- `-t, --template string`: Template to use (approval, fraud, inventory, payment, customer, blank) (default: "blank")
- `-o, --output string`: Output file name (default: `<workflow-name>.json`)

**Available Templates:**
- **approval**: Order approval workflow for high-value transactions
- **fraud**: Fraud detection and prevention workflow
- **inventory**: Low inventory alert and replenishment workflow
- **payment**: Payment failure handling workflow
- **customer**: New customer onboarding workflow
- **blank**: Empty workflow template

**Examples:**

```bash
# Create approval workflow
workflow init my-approval-workflow --template approval

# Create custom workflow with specific output file
workflow init custom-workflow --template blank --output custom.json

# Create fraud detection workflow
workflow init fraud-check --template fraud
```

**Output:**
```
✅ Created workflow 'my-approval-workflow' from template 'approval'
📄 File: my-approval-workflow.json

Next steps:
  1. Edit the workflow: my-approval-workflow.json
  2. Validate: workflow validate my-approval-workflow.json
  3. Deploy: workflow deploy my-approval-workflow.json
```

---

### `workflow validate` - Validate a Workflow Definition

Validate a workflow definition file to ensure it meets all requirements.

**Usage:**
```bash
workflow validate [workflow-file]
```

**The validator checks:**
- Required fields (workflow_id, name, version)
- Valid trigger types
- Step structure and configuration
- Condition and action syntax
- Reference consistency

**Examples:**

```bash
# Validate a workflow file
workflow validate workflow.json

# Validate with JSON output
workflow validate approval-workflow.json --json
```

**Output:**
```
🔍 Validating workflow: workflow.json

✅ Workflow is valid!

Next step:
  workflow deploy workflow.json
```

**Error Output:**
```
🔍 Validating workflow: workflow.json

❌ Workflow validation failed with 2 error(s):

  1. workflow_id is required
  2. step[0].action is required for action type

💡 Tip: Fix the errors above and run validate again
```

---

### `workflow deploy` - Deploy a Workflow to the Server

Deploy a workflow definition to the workflow engine server.

**Usage:**
```bash
workflow deploy [workflow-file] [flags]
```

**Flags:**
- `-u, --update`: Update workflow if it already exists

**The deploy command will:**
1. Validate the workflow definition
2. Check if the API server is reachable
3. Create or update the workflow on the server
4. Enable the workflow (if not disabled in definition)

**Examples:**

```bash
# Deploy a new workflow
workflow deploy workflow.json

# Update an existing workflow
workflow deploy workflow.json --update

# Deploy to a specific API server
workflow deploy workflow.json --api-url http://prod.example.com:8080
```

**Output:**
```
🔍 Validating workflow...
✅ Validation passed
🔗 Connecting to API: http://localhost:8080
🚀 Deploying workflow 'Order Approval Workflow'...
✅ Workflow deployed successfully!

📦 Workflow Details:
  ID:         abc123-def456-...
  Workflow:   order-approval-workflow
  Name:       Order Approval Workflow
  Version:    1.0.0
  Enabled:    true
  Created:    2025-11-05 10:30:45
  Updated:    2025-11-05 10:30:45

📋 Next steps:
  • List workflows:   workflow list
  • Test workflow:    workflow test workflow.json --event event.json
  • View logs:        workflow logs <execution-id>
```

---

### `workflow list` - List All Workflows

List all workflows from the workflow engine server.

**Usage:**
```bash
workflow list [flags]
```

**Flags:**
- `--enabled-only`: Show only enabled workflows
- `--disabled-only`: Show only disabled workflows

**Examples:**

```bash
# List all workflows
workflow list

# List only enabled workflows
workflow list --enabled-only

# Get JSON output
workflow list --json
```

**Output:**
```
📋 Found 3 workflow(s):

┌────────────────────────────────────────┬──────────────────────────┬─────────┬─────────┐
│ Workflow ID                            │ Name                     │ Version │ Status  │
├────────────────────────────────────────┼──────────────────────────┼─────────┼─────────┤
│ order-approval-workflow                │ Order Approval Workflow  │ 1.0.0   │ ✅ Enabled │
│ fraud-detection-workflow               │ Fraud Detection          │ 1.0.0   │ ✅ Enabled │
│ inventory-alert-workflow               │ Inventory Alert          │ 1.0.0   │ ❌ Disabled│
└────────────────────────────────────────┴──────────────────────────┴─────────┴─────────┘

📖 View details:
  workflow logs <execution-id>
```

---

### `workflow test` - Test a Workflow with Sample Data

Test a workflow by sending a test event to the workflow engine.

**Usage:**
```bash
workflow test [workflow-file] [flags]
```

**Flags:**
- `-e, --event string`: Event data file (JSON)
- `-w, --wait`: Wait for execution results
- `-t, --timeout int`: Timeout in seconds when waiting (default: 30)

**The test command will:**
1. Load the workflow definition
2. Load the event data from file or use a sample event
3. Send the event to the API
4. Optionally wait for execution results

**Examples:**

```bash
# Test with a sample event (auto-generated based on trigger)
workflow test workflow.json

# Test with a custom event file
workflow test workflow.json --event test-event.json

# Test and wait for results
workflow test workflow.json --event order.json --wait
```

**Sample Event File** (`test-event.json`):
```json
{
  "event_type": "order.created",
  "source": "test",
  "payload": {
    "order": {
      "id": "order-123",
      "customer_id": "cust-456",
      "total": 1500.00,
      "items": [
        {
          "product_id": "prod-789",
          "quantity": 2,
          "price": 750.00
        }
      ]
    }
  }
}
```

**Output:**
```
🔗 Connecting to API...
🚀 Sending test event (type: order.created)...
✅ Event sent successfully!

📋 Event Details:
  ID:        abc123-def456-...
  Type:      order.created
  Timestamp: 2025-11-05 10:35:22

💡 Next steps:
  • View executions: workflow logs
```

---

### `workflow logs` - View Workflow Execution Logs

View execution logs for workflows. Can show a specific execution or list recent executions.

**Usage:**
```bash
workflow logs [execution-id] [flags]
```

**Flags:**
- `--workflow string`: Filter by workflow ID
- `--status string`: Filter by status (pending, running, completed, failed)
- `--limit int`: Number of executions to show (default: 20)
- `-f, --follow`: Follow execution logs in real-time (requires execution ID)

**Examples:**

```bash
# List recent executions
workflow logs

# Show specific execution
workflow logs abc123-def456-...

# Filter by workflow
workflow logs --workflow order-approval-workflow

# Filter by status
workflow logs --status completed

# Show last 50 executions
workflow logs --limit 50
```

**List Output:**
```
📋 Fetching executions...

📋 Found 3 execution(s):

┌──────────────────────────────────────┬──────────────────────────┬────────────┬─────────────────────┐
│ Execution ID                         │ Workflow                 │ Status     │ Started At          │
├──────────────────────────────────────┼──────────────────────────┼────────────┼─────────────────────┤
│ abc123-def456-789012-345678          │ order-approval-workflow  │ ✅ Completed│ 2025-11-05 10:35:22 │
│ def456-789012-345678-abc123          │ fraud-detection-workflow │ 🏃 Running  │ 2025-11-05 10:36:15 │
│ 789012-345678-abc123-def456          │ inventory-alert-workflow │ ❌ Failed   │ 2025-11-05 10:37:08 │
└──────────────────────────────────────┴──────────────────────────┴────────────┴─────────────────────┘

📖 View details:
  workflow logs <execution-id>
```

**Detail Output:**
```
🔍 Fetching execution: abc123-def456-789012-345678

📊 Execution Details
═══════════════════════════════════════════════════════════
ID:           abc123-def456-789012-345678
Workflow:     order-approval-workflow
Status:       ✅ Completed
Started:      2025-11-05 10:35:22
Completed:    2025-11-05 10:35:24
Duration:     2.15s
Error:

📝 Execution Metadata:
───────────────────────────────────────────────────────────
{
  "steps_executed": 3,
  "decision": "approved",
  "approver": "manager@example.com"
}

📦 Context:
───────────────────────────────────────────────────────────
{
  "order_value": 1500.00,
  "customer_id": "cust-456",
  "approval_required": true
}
```

---

## Global Flags

These flags are available for all commands:

- `--api-url string`: Workflow API URL (default: "http://localhost:8080")
- `--api-token string`: API authentication token
- `--config string`: Config file (default: `$HOME/.workflow-cli.yaml`)
- `--json`: Output results in JSON format
- `-h, --help`: Help for any command

## Common Workflows

### Creating and Deploying a Workflow

```bash
# 1. Initialize from template
workflow init my-workflow --template approval

# 2. Edit the generated file
vim my-workflow.json

# 3. Validate the workflow
workflow validate my-workflow.json

# 4. Deploy to server
workflow deploy my-workflow.json

# 5. Test the workflow
workflow test my-workflow.json
```

### Testing and Monitoring

```bash
# 1. Test with custom event
workflow test workflow.json --event test-event.json

# 2. View recent executions
workflow logs

# 3. View specific execution details
workflow logs <execution-id>

# 4. Filter by workflow
workflow logs --workflow my-workflow --limit 50
```

### Production Deployment

```bash
# 1. Deploy to production server
workflow deploy workflow.json \
  --api-url https://workflows.prod.example.com \
  --api-token $PROD_API_TOKEN

# 2. Verify deployment
workflow list \
  --api-url https://workflows.prod.example.com \
  --api-token $PROD_API_TOKEN

# 3. Monitor executions
workflow logs --workflow my-workflow \
  --api-url https://workflows.prod.example.com \
  --api-token $PROD_API_TOKEN
```

## Troubleshooting

### Connection Errors

If you see "API health check failed":
1. Check if the API server is running
2. Verify the API URL is correct
3. Check network connectivity

```bash
# Test API connectivity
curl http://localhost:8080/health
```

### Validation Errors

If validation fails:
1. Review the error messages carefully
2. Check the workflow definition against examples
3. Ensure all required fields are present
4. Verify trigger types and step configurations

### Authentication Errors

If you see "401 Unauthorized":
1. Ensure you have a valid API token
2. Set the token via config file, environment variable, or flag
3. Contact your administrator for a new token

## Examples Directory

See the `templates/workflows/` directory for example workflow definitions:
- `approval.json` - Order approval workflow
- `fraud.json` - Fraud detection workflow
- `inventory.json` - Inventory management workflow
- `payment.json` - Payment failure handling
- `customer.json` - Customer onboarding

## Support

For more information:
- See the main project README
- Check the API documentation
- Review the workflow definition guide
- Contact the development team

## Version

CLI Tool Version: 1.0.0
Compatible with API: v1

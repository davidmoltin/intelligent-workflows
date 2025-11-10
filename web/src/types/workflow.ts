export interface Workflow {
  id: string
  workflow_id: string
  version: string
  name: string
  description?: string
  definition: WorkflowDefinition
  enabled: boolean
  created_at: string
  updated_at: string
  tags?: string[]
}

export interface WorkflowDefinition {
  workflow_id: string
  version: string
  name: string
  description?: string
  enabled: boolean
  trigger: Trigger
  context?: ContextConfig
  steps: Step[]
}

export interface Trigger {
  type: 'event' | 'schedule' | 'manual'
  event?: string
  schedule?: string
}

export interface ContextConfig {
  load?: string[]
}

export interface Step {
  id: string
  type: 'condition' | 'action' | 'execute' | 'parallel' | 'foreach'
  name?: string
  condition?: Condition
  action?: Action
  execute?: ExecuteAction[]
  parallel?: ParallelStep
  on_true?: string
  on_false?: string
  next?: string
  metadata?: Record<string, any>
  retry?: RetryConfig
  // Legacy fields for backwards compatibility
  steps?: Step[]
  strategy?: string
}

export interface Condition {
  field?: string
  operator?: 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'contains'
  value?: any
  and?: Condition[]
  or?: Condition[]
  not?: Condition
}

export interface Action {
  type: 'allow' | 'block' | 'execute' | 'wait' | 'require_approval'
  reason?: string
}

export interface ExecuteAction {
  type: 'notify' | 'webhook' | 'http_request' | 'create_record' | 'update_record' | 'create_approval_request' | 'log'
  // Notify action fields
  recipients?: string[]
  message?: string
  // Webhook/HTTP request fields
  url?: string
  method?: string
  headers?: Record<string, string>
  body?: Record<string, any>
  // Record action fields
  entity?: string
  entity_id?: string
  data?: Record<string, any>
}

export interface ParallelStep {
  steps: Step[]
  strategy: 'all_must_pass' | 'any_can_pass' | 'best_effort'
}

export interface RetryConfig {
  max_attempts?: number
  backoff?: string
  retry_on?: string[]
}

export interface Execution {
  id: string
  execution_id: string
  workflow_id: string
  trigger_event: string
  trigger_payload: Record<string, any>
  context?: Record<string, any>
  status: ExecutionStatus
  result?: ExecutionResult
  started_at: string
  completed_at?: string
  duration_ms?: number
  error_message?: string
  steps?: StepExecution[]
}

export type ExecutionStatus = 'pending' | 'running' | 'waiting' | 'completed' | 'failed' | 'blocked' | 'cancelled' | 'paused'
export type ExecutionResult = 'allowed' | 'blocked' | 'executed'

export interface StepExecution {
  id: string
  execution_id: string
  step_id: string
  step_type: string
  status: ExecutionStatus
  input?: Record<string, any>
  output?: Record<string, any>
  result?: any
  started_at: string
  completed_at?: string
  duration_ms?: number
  error_message?: string
}

export interface ExecutionTraceResponse {
  execution: Execution
  steps: StepExecution[]
  workflow?: Workflow
}

export interface ApprovalRequest {
  id: string
  request_id: string
  execution_id: string
  entity_type: string
  entity_id: string
  requester_id: string
  approver_role: string
  approver_id?: string
  status: ApprovalStatus
  reason: string
  decision_reason?: string
  requested_at: string
  decided_at?: string
  expires_at?: string
}

export type ApprovalStatus = 'pending' | 'approved' | 'rejected' | 'expired'

export interface CreateWorkflowRequest {
  workflow_id: string
  version: string
  name: string
  description?: string
  definition: WorkflowDefinition
  tags?: string[]
}

export interface UpdateWorkflowRequest {
  name?: string
  description?: string
  definition?: WorkflowDefinition
  tags?: string[]
}

export interface IngestEventRequest {
  event_type: string
  source: string
  payload: Record<string, any>
}

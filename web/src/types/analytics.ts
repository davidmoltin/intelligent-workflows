export interface ExecutionStats {
  total_executions: number
  completed: number
  failed: number
  running: number
  pending: number
  blocked: number
  cancelled: number
  paused: number
  success_rate: number
  failure_rate: number
  avg_duration_ms: number
  min_duration_ms: number
  max_duration_ms: number
}

export interface ExecutionTrend {
  timestamp: string
  total: number
  completed: number
  failed: number
  running: number
}

export interface WorkflowStats {
  workflow_id: string
  workflow_name: string
  total_executions: number
  completed: number
  failed: number
  success_rate: number
  avg_duration_ms: number
}

export interface ExecutionError {
  execution_id: string
  workflow_id: string
  workflow_name: string
  execution_id_str: string
  error_message: string
  started_at: string
  completed_at?: string
}

export interface StepStats {
  step_id: string
  step_type: string
  total_executions: number
  completed: number
  failed: number
  success_rate: number
  avg_duration_ms: number
}

export interface AnalyticsDashboard {
  stats: ExecutionStats
  trends: ExecutionTrend[]
  workflow_stats: WorkflowStats[]
  recent_errors: ExecutionError[]
  step_stats?: StepStats[]
  time_range: string
  generated_at: string
}

export type TimeRange = '1h' | '24h' | '7d' | '30d'

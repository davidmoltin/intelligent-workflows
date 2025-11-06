import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useExecutionTrace } from '@/api/hooks'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ArrowLeft, ChevronDown, ChevronRight, Clock, AlertCircle, CheckCircle2, Loader2 } from 'lucide-react'
import type { StepExecution } from '@/types/workflow'

export function ExecutionDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: trace, isLoading, error } = useExecutionTrace(id!)
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set())

  const toggleStep = (stepId: string) => {
    const newExpanded = new Set(expandedSteps)
    if (newExpanded.has(stepId)) {
      newExpanded.delete(stepId)
    } else {
      newExpanded.add(stepId)
    }
    setExpandedSteps(newExpanded)
  }

  const getStatusVariant = (status: string) => {
    switch (status) {
      case 'completed':
        return 'default'
      case 'running':
        return 'secondary'
      case 'failed':
        return 'destructive'
      case 'blocked':
        return 'outline'
      case 'skipped':
        return 'outline'
      default:
        return 'outline'
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle2 className="h-5 w-5 text-green-500" />
      case 'running':
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />
      case 'failed':
        return <AlertCircle className="h-5 w-5 text-red-500" />
      default:
        return <Clock className="h-5 w-5 text-gray-500" />
    }
  }

  const formatDuration = (ms?: number) => {
    if (!ms) return '-'
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(2)}s`
  }

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString()
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading execution trace...</div>
      </div>
    )
  }

  if (error || !trace) {
    return (
      <div className="flex h-full items-center justify-center">
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle>Error Loading Execution</CardTitle>
            <CardDescription>
              {error instanceof Error ? error.message : 'Execution not found'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => navigate('/executions')}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Executions
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  const { execution, steps, workflow } = trace

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate('/executions')}
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Execution Trace</h1>
            <p className="text-muted-foreground font-mono text-sm">
              {execution.execution_id}
            </p>
          </div>
        </div>
        <Badge variant={getStatusVariant(execution.status)} className="text-sm">
          {execution.status}
        </Badge>
      </div>

      {/* Execution Overview */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Workflow</CardDescription>
            <CardTitle className="text-lg">
              {workflow ? (
                <Link
                  to={`/workflows/${execution.workflow_id}`}
                  className="text-primary hover:underline"
                >
                  {workflow.name}
                </Link>
              ) : (
                execution.workflow_id
              )}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Duration</CardDescription>
            <CardTitle className="text-2xl">
              {formatDuration(execution.duration_ms)}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Steps</CardDescription>
            <CardTitle className="text-2xl">{steps.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Result</CardDescription>
            <CardTitle className="text-lg">
              {execution.result ? (
                <Badge variant={getStatusVariant(execution.status)}>
                  {execution.result}
                </Badge>
              ) : (
                '-'
              )}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      {/* Execution Details */}
      <Card>
        <CardHeader>
          <CardTitle>Execution Details</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Trigger Event
              </p>
              <p className="text-sm font-mono">{execution.trigger_event}</p>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Started At
              </p>
              <p className="text-sm">{formatTimestamp(execution.started_at)}</p>
            </div>
            {execution.completed_at && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">
                  Completed At
                </p>
                <p className="text-sm">
                  {formatTimestamp(execution.completed_at)}
                </p>
              </div>
            )}
            {execution.error_message && (
              <div className="md:col-span-2">
                <p className="text-sm font-medium text-destructive">Error</p>
                <p className="text-sm text-destructive font-mono mt-1">
                  {execution.error_message}
                </p>
              </div>
            )}
          </div>

          {/* Trigger Payload */}
          {execution.trigger_payload && Object.keys(execution.trigger_payload).length > 0 && (
            <div>
              <p className="text-sm font-medium text-muted-foreground mb-2">
                Trigger Payload
              </p>
              <pre className="bg-muted p-3 rounded-md text-xs overflow-x-auto">
                {JSON.stringify(execution.trigger_payload, null, 2)}
              </pre>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Step Execution Timeline */}
      <Card>
        <CardHeader>
          <CardTitle>Execution Steps</CardTitle>
          <CardDescription>
            Step-by-step execution flow with detailed information
          </CardDescription>
        </CardHeader>
        <CardContent>
          {steps.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">No steps executed</p>
            </div>
          ) : (
            <div className="space-y-3">
              {steps.map((step, index) => (
                <StepCard
                  key={step.id}
                  step={step}
                  index={index}
                  isExpanded={expandedSteps.has(step.id)}
                  onToggle={() => toggleStep(step.id)}
                  getStatusVariant={getStatusVariant}
                  getStatusIcon={getStatusIcon}
                  formatDuration={formatDuration}
                  formatTimestamp={formatTimestamp}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

interface StepCardProps {
  step: StepExecution
  index: number
  isExpanded: boolean
  onToggle: () => void
  getStatusVariant: (status: string) => any
  getStatusIcon: (status: string) => React.ReactNode
  formatDuration: (ms?: number) => string
  formatTimestamp: (timestamp: string) => string
}

function StepCard({
  step,
  index,
  isExpanded,
  onToggle,
  getStatusVariant,
  getStatusIcon,
  formatDuration,
  formatTimestamp,
}: StepCardProps) {
  return (
    <div className="border rounded-lg">
      <button
        onClick={onToggle}
        className="w-full p-4 flex items-center gap-4 hover:bg-muted/50 transition-colors"
      >
        {/* Step Number and Icon */}
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-sm font-medium">
            {index + 1}
          </div>
          {getStatusIcon(step.status)}
        </div>

        {/* Step Info */}
        <div className="flex-1 text-left">
          <div className="flex items-center gap-2">
            <p className="font-medium">{step.step_id}</p>
            <Badge variant="outline" className="text-xs">
              {step.step_type}
            </Badge>
            <Badge variant={getStatusVariant(step.status)} className="text-xs">
              {step.status}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Duration: {formatDuration(step.duration_ms)}
          </p>
        </div>

        {/* Expand Icon */}
        {isExpanded ? (
          <ChevronDown className="h-5 w-5 text-muted-foreground" />
        ) : (
          <ChevronRight className="h-5 w-5 text-muted-foreground" />
        )}
      </button>

      {/* Expanded Content */}
      {isExpanded && (
        <div className="border-t p-4 space-y-4 bg-muted/20">
          {/* Timestamps */}
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                Started At
              </p>
              <p className="text-sm">{formatTimestamp(step.started_at)}</p>
            </div>
            {step.completed_at && (
              <div>
                <p className="text-sm font-medium text-muted-foreground">
                  Completed At
                </p>
                <p className="text-sm">{formatTimestamp(step.completed_at)}</p>
              </div>
            )}
          </div>

          {/* Error Message */}
          {step.error_message && (
            <div>
              <p className="text-sm font-medium text-destructive mb-2">
                Error Message
              </p>
              <pre className="bg-destructive/10 border border-destructive p-3 rounded-md text-xs overflow-x-auto text-destructive">
                {step.error_message}
              </pre>
            </div>
          )}

          {/* Input */}
          {step.input && Object.keys(step.input).length > 0 && (
            <div>
              <p className="text-sm font-medium text-muted-foreground mb-2">
                Input
              </p>
              <pre className="bg-muted p-3 rounded-md text-xs overflow-x-auto">
                {JSON.stringify(step.input, null, 2)}
              </pre>
            </div>
          )}

          {/* Output */}
          {step.output && Object.keys(step.output).length > 0 && (
            <div>
              <p className="text-sm font-medium text-muted-foreground mb-2">
                Output
              </p>
              <pre className="bg-muted p-3 rounded-md text-xs overflow-x-auto">
                {JSON.stringify(step.output, null, 2)}
              </pre>
            </div>
          )}

          {/* Result (if different from output) */}
          {step.result && (
            <div>
              <p className="text-sm font-medium text-muted-foreground mb-2">
                Result
              </p>
              <pre className="bg-muted p-3 rounded-md text-xs overflow-x-auto">
                {JSON.stringify(step.result, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

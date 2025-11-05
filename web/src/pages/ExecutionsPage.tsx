import { Link } from 'react-router-dom'
import { useExecutions } from '@/api/hooks'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Eye } from 'lucide-react'

export function ExecutionsPage() {
  const { data: executions, isLoading, error } = useExecutions()

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
      default:
        return 'outline'
    }
  }

  const getResultVariant = (result?: string) => {
    switch (result) {
      case 'allowed':
        return 'default'
      case 'blocked':
        return 'destructive'
      case 'executed':
        return 'secondary'
      default:
        return 'outline'
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading executions...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle>Error Loading Executions</CardTitle>
            <CardDescription>
              {error instanceof Error ? error.message : 'An error occurred'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Executions</h1>
        <p className="text-muted-foreground">
          Monitor workflow execution history and performance
        </p>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total Executions</CardDescription>
            <CardTitle className="text-3xl">
              {executions?.length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Completed</CardDescription>
            <CardTitle className="text-3xl">
              {executions?.filter((e) => e.status === 'completed').length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Failed</CardDescription>
            <CardTitle className="text-3xl">
              {executions?.filter((e) => e.status === 'failed').length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Running</CardDescription>
            <CardTitle className="text-3xl">
              {executions?.filter((e) => e.status === 'running').length || 0}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      {/* Executions Table */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Executions</CardTitle>
          <CardDescription>
            View detailed execution history and traces
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!executions || executions.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-muted-foreground">No executions found</p>
              <p className="mt-2 text-sm text-muted-foreground">
                Workflows will appear here once they start executing
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Execution ID</TableHead>
                  <TableHead>Workflow</TableHead>
                  <TableHead>Trigger Event</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Result</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Started At</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {executions.map((execution) => (
                  <TableRow key={execution.id}>
                    <TableCell className="font-mono text-xs">
                      {execution.execution_id}
                    </TableCell>
                    <TableCell>
                      <Link
                        to={`/workflows/${execution.workflow_id}`}
                        className="text-primary hover:underline"
                      >
                        View Workflow
                      </Link>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {execution.trigger_event}
                    </TableCell>
                    <TableCell>
                      <Badge variant={getStatusVariant(execution.status)}>
                        {execution.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {execution.result && (
                        <Badge variant={getResultVariant(execution.result)}>
                          {execution.result}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {execution.duration_ms
                        ? `${execution.duration_ms}ms`
                        : '-'}
                    </TableCell>
                    <TableCell>
                      {new Date(execution.started_at).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Link to={`/executions/${execution.id}`}>
                        <Button variant="ghost" size="icon">
                          <Eye className="h-4 w-4" />
                        </Button>
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

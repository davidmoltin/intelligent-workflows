import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  useWorkflow,
  useDeleteWorkflow,
  useEnableWorkflow,
  useDisableWorkflow,
} from '@/api/hooks'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { ArrowLeft, Edit, Trash2, List, Network } from 'lucide-react'
import { DeleteWorkflowDialog } from '@/components/DeleteWorkflowDialog'
import { WorkflowGraphVisualizer } from '@/components/WorkflowGraphVisualizer'

export function WorkflowDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: workflow, isLoading, error } = useWorkflow(id!)
  const deleteWorkflow = useDeleteWorkflow()
  const enableWorkflow = useEnableWorkflow()
  const disableWorkflow = useDisableWorkflow()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)

  const handleToggleEnabled = async () => {
    if (!workflow) return

    try {
      if (workflow.enabled) {
        await disableWorkflow.mutateAsync(workflow.id)
      } else {
        await enableWorkflow.mutateAsync(workflow.id)
      }
    } catch (error) {
      console.error('Failed to toggle workflow:', error)
    }
  }

  const handleDelete = async () => {
    if (!workflow) return

    try {
      await deleteWorkflow.mutateAsync(workflow.id)
      navigate('/workflows')
    } catch (error) {
      console.error('Failed to delete workflow:', error)
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading workflow...</div>
      </div>
    )
  }

  if (error || !workflow) {
    return (
      <div className="flex h-full items-center justify-center">
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle>Error Loading Workflow</CardTitle>
            <CardDescription>
              {error instanceof Error ? error.message : 'Workflow not found'}
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate('/workflows')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{workflow.name}</h1>
            <p className="text-muted-foreground">{workflow.description}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <Link to={`/workflows/${workflow.id}/edit`}>
            <Button variant="outline">
              <Edit className="mr-2 h-4 w-4" />
              Edit
            </Button>
          </Link>
          <Button
            variant="destructive"
            onClick={() => setShowDeleteDialog(true)}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
        </div>
      </div>

      {/* Status Card */}
      <Card>
        <CardHeader>
          <CardTitle>Status</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div className="space-y-2">
              <Label htmlFor="enabled-toggle">
                {workflow.enabled ? 'Enabled' : 'Disabled'}
              </Label>
              <p className="text-sm text-muted-foreground">
                {workflow.enabled
                  ? 'This workflow is currently active and will execute on triggers'
                  : 'This workflow is currently disabled and will not execute'}
              </p>
            </div>
            <Switch
              id="enabled-toggle"
              checked={workflow.enabled}
              onCheckedChange={handleToggleEnabled}
              disabled={
                enableWorkflow.isPending || disableWorkflow.isPending
              }
            />
          </div>
        </CardContent>
      </Card>

      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <Label className="text-muted-foreground">Workflow ID</Label>
              <p className="font-mono">{workflow.workflow_id}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Version</Label>
              <p>{workflow.version}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Created</Label>
              <p>{new Date(workflow.created_at).toLocaleString()}</p>
            </div>
            <div>
              <Label className="text-muted-foreground">Last Updated</Label>
              <p>{new Date(workflow.updated_at).toLocaleString()}</p>
            </div>
          </div>
          {workflow.tags && workflow.tags.length > 0 && (
            <div>
              <Label className="text-muted-foreground">Tags</Label>
              <div className="flex gap-2 mt-2">
                {workflow.tags.map((tag) => (
                  <Badge key={tag} variant="secondary">
                    {tag}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Trigger Configuration */}
      <Card>
        <CardHeader>
          <CardTitle>Trigger Configuration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label className="text-muted-foreground">Trigger Type</Label>
            <div className="mt-2">
              <Badge>{workflow.definition.trigger.type}</Badge>
            </div>
          </div>
          {workflow.definition.trigger.event && (
            <div>
              <Label className="text-muted-foreground">Event Type</Label>
              <p className="font-mono">{workflow.definition.trigger.event}</p>
            </div>
          )}
          {workflow.definition.trigger.schedule && (
            <div>
              <Label className="text-muted-foreground">Schedule (Cron)</Label>
              <p className="font-mono">{workflow.definition.trigger.schedule}</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Steps */}
      <Card>
        <CardHeader>
          <CardTitle>Workflow Steps</CardTitle>
          <CardDescription>
            {workflow.definition.steps.length} step(s) defined
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="graph" className="w-full">
            <TabsList className="grid w-full max-w-md grid-cols-2">
              <TabsTrigger value="graph" className="flex items-center gap-2">
                <Network className="h-4 w-4" />
                Graph View
              </TabsTrigger>
              <TabsTrigger value="list" className="flex items-center gap-2">
                <List className="h-4 w-4" />
                List View
              </TabsTrigger>
            </TabsList>
            <TabsContent value="graph" className="mt-4">
              <div className="h-[600px] w-full">
                <WorkflowGraphVisualizer
                  trigger={workflow.definition.trigger}
                  steps={workflow.definition.steps}
                  readonly={true}
                />
              </div>
            </TabsContent>
            <TabsContent value="list" className="mt-4">
              <div className="space-y-4">
                {workflow.definition.steps.map((step, index) => (
                  <Card key={step.id}>
                    <CardHeader className="pb-2">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Badge>{step.type}</Badge>
                          <CardTitle className="text-base">
                            {step.name || step.id}
                          </CardTitle>
                        </div>
                        <span className="text-sm text-muted-foreground">
                          Step {index + 1}
                        </span>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="text-sm space-y-1">
                        <div className="text-muted-foreground">ID: {step.id}</div>
                        {step.next && (
                          <div className="text-muted-foreground">
                            Next: {step.next}
                          </div>
                        )}
                        {step.type === 'condition' && step.condition && (
                          <div className="mt-2 p-2 bg-muted rounded-md">
                            <div className="font-medium">Condition:</div>
                            <div className="font-mono text-xs">
                              {step.condition.field} {step.condition.operator}{' '}
                              {JSON.stringify(step.condition.value)}
                            </div>
                            {step.on_true && (
                              <div className="text-muted-foreground">
                                On True: {step.on_true}
                              </div>
                            )}
                            {step.on_false && (
                              <div className="text-muted-foreground">
                                On False: {step.on_false}
                              </div>
                            )}
                          </div>
                        )}
                        {step.type === 'action' && step.action && (
                          <div className="mt-2 p-2 bg-muted rounded-md">
                            <div className="font-medium">Action:</div>
                            <div>{step.action.type}</div>
                            {step.action.reason && (
                              <div className="text-muted-foreground text-xs">
                                {step.action.reason}
                              </div>
                            )}
                          </div>
                        )}
                        {step.type === 'execute' && step.execute && (
                          <div className="mt-2 p-2 bg-muted rounded-md">
                            <div className="font-medium">Execute Actions:</div>
                            {step.execute.map((action, i) => (
                              <div key={i} className="text-xs">
                                {action.type}
                                {action.config && (
                                  <pre className="mt-1 text-xs overflow-auto">
                                    {JSON.stringify(action.config, null, 2)}
                                  </pre>
                                )}
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>

      {/* Delete Dialog */}
      <DeleteWorkflowDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        workflowName={workflow.name}
        onConfirm={handleDelete}
        isDeleting={deleteWorkflow.isPending}
      />
    </div>
  )
}

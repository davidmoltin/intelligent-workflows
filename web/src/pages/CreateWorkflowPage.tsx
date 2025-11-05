import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useCreateWorkflow } from '@/api/hooks'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ArrowLeft } from 'lucide-react'
import type { WorkflowDefinition, Step } from '@/types/workflow'
import { StepBuilder } from '@/components/StepBuilder'

export function CreateWorkflowPage() {
  const navigate = useNavigate()
  const createWorkflow = useCreateWorkflow()

  const [workflowId, setWorkflowId] = useState('')
  const [version, setVersion] = useState('1.0.0')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [triggerType, setTriggerType] = useState<'event' | 'schedule' | 'manual'>('event')
  const [eventType, setEventType] = useState('')
  const [schedule, setSchedule] = useState('')
  const [steps, setSteps] = useState<Step[]>([])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const definition: WorkflowDefinition = {
      workflow_id: workflowId,
      version,
      name,
      description,
      enabled: true,
      trigger: {
        type: triggerType,
        ...(triggerType === 'event' && { event: eventType }),
        ...(triggerType === 'schedule' && { schedule }),
      },
      steps,
    }

    try {
      await createWorkflow.mutateAsync({
        workflow_id: workflowId,
        version,
        name,
        description,
        definition,
      })
      navigate('/workflows')
    } catch (error) {
      console.error('Failed to create workflow:', error)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/workflows')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Create Workflow</h1>
          <p className="text-muted-foreground">
            Define a new intelligent workflow
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Information */}
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
            <CardDescription>
              Configure the basic details of your workflow
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="workflow-id">Workflow ID</Label>
                <Input
                  id="workflow-id"
                  placeholder="e.g., order-approval"
                  value={workflowId}
                  onChange={(e) => setWorkflowId(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="version">Version</Label>
                <Input
                  id="version"
                  placeholder="e.g., 1.0.0"
                  value={version}
                  onChange={(e) => setVersion(e.target.value)}
                  required
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="e.g., High Value Order Approval"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                placeholder="Describe what this workflow does..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
              />
            </div>
          </CardContent>
        </Card>

        {/* Trigger Configuration */}
        <Card>
          <CardHeader>
            <CardTitle>Trigger Configuration</CardTitle>
            <CardDescription>
              Define when this workflow should execute
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="trigger-type">Trigger Type</Label>
              <Select
                value={triggerType}
                onValueChange={(value: any) => setTriggerType(value)}
              >
                <SelectTrigger id="trigger-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="event">Event</SelectItem>
                  <SelectItem value="schedule">Schedule</SelectItem>
                  <SelectItem value="manual">Manual</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {triggerType === 'event' && (
              <div className="space-y-2">
                <Label htmlFor="event-type">Event Type</Label>
                <Input
                  id="event-type"
                  placeholder="e.g., order.created"
                  value={eventType}
                  onChange={(e) => setEventType(e.target.value)}
                  required
                />
              </div>
            )}

            {triggerType === 'schedule' && (
              <div className="space-y-2">
                <Label htmlFor="schedule">Schedule (Cron)</Label>
                <Input
                  id="schedule"
                  placeholder="e.g., 0 0 * * *"
                  value={schedule}
                  onChange={(e) => setSchedule(e.target.value)}
                  required
                />
              </div>
            )}
          </CardContent>
        </Card>

        {/* Step Builder */}
        <Card>
          <CardHeader>
            <CardTitle>Workflow Steps</CardTitle>
            <CardDescription>
              Define the steps that make up your workflow
            </CardDescription>
          </CardHeader>
          <CardContent>
            <StepBuilder steps={steps} onChange={setSteps} />
          </CardContent>
        </Card>

        {/* Actions */}
        <div className="flex justify-end gap-4">
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate('/workflows')}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={createWorkflow.isPending}>
            {createWorkflow.isPending ? 'Creating...' : 'Create Workflow'}
          </Button>
        </div>
      </form>
    </div>
  )
}

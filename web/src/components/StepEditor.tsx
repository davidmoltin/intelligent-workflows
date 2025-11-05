import { useState } from 'react'
import type { Step, Condition, Action, ExecuteAction } from '@/types/workflow'
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
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface StepEditorProps {
  step: Step | null
  onSave: (step: Step) => void
  onCancel: () => void
}

export function StepEditor({ step, onSave, onCancel }: StepEditorProps) {
  const [stepId, setStepId] = useState(step?.id || '')
  const [stepName, setStepName] = useState(step?.name || '')
  const [stepType, setStepType] = useState<Step['type']>(step?.type || 'condition')
  const [nextStep, setNextStep] = useState(step?.next || '')

  // Condition fields
  const [conditionField, setConditionField] = useState(step?.condition?.field || '')
  const [conditionOperator, setConditionOperator] = useState<Condition['operator']>(
    step?.condition?.operator || 'eq'
  )
  const [conditionValue, setConditionValue] = useState(
    step?.condition?.value ? JSON.stringify(step.condition.value) : ''
  )
  const [onTrue, setOnTrue] = useState(step?.on_true || '')
  const [onFalse, setOnFalse] = useState(step?.on_false || '')

  // Action fields
  const [actionType, setActionType] = useState<Action['action']>(
    step?.action?.action || 'allow'
  )
  const [actionReason, setActionReason] = useState(step?.action?.reason || '')

  // Execute fields
  const [executeType, setExecuteType] = useState<ExecuteAction['type']>('notify')
  const [executeConfig, setExecuteConfig] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const newStep: Step = {
      id: stepId,
      type: stepType,
      name: stepName || undefined,
      next: nextStep || undefined,
    }

    if (stepType === 'condition') {
      newStep.condition = {
        field: conditionField,
        operator: conditionOperator,
        value: conditionValue ? JSON.parse(conditionValue) : undefined,
      }
      newStep.on_true = onTrue || undefined
      newStep.on_false = onFalse || undefined
    } else if (stepType === 'action') {
      newStep.action = {
        action: actionType,
        reason: actionReason || undefined,
      }
    } else if (stepType === 'execute') {
      newStep.execute = [
        {
          type: executeType,
          config: executeConfig ? JSON.parse(executeConfig) : undefined,
        },
      ]
    }

    onSave(newStep)
  }

  return (
    <Card className="border-2 border-primary">
      <CardHeader>
        <CardTitle>{step ? 'Edit Step' : 'New Step'}</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="step-id">Step ID</Label>
              <Input
                id="step-id"
                placeholder="e.g., check-amount"
                value={stepId}
                onChange={(e) => setStepId(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="step-name">Step Name</Label>
              <Input
                id="step-name"
                placeholder="e.g., Check Order Amount"
                value={stepName}
                onChange={(e) => setStepName(e.target.value)}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="step-type">Step Type</Label>
            <Select
              value={stepType}
              onValueChange={(value: any) => setStepType(value)}
            >
              <SelectTrigger id="step-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="condition">Condition</SelectItem>
                <SelectItem value="action">Action</SelectItem>
                <SelectItem value="execute">Execute</SelectItem>
                <SelectItem value="parallel">Parallel</SelectItem>
                <SelectItem value="foreach">For Each</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {stepType === 'condition' && (
            <div className="space-y-4 rounded-lg border p-4">
              <h4 className="font-medium">Condition Configuration</h4>
              <div className="space-y-2">
                <Label htmlFor="condition-field">Field</Label>
                <Input
                  id="condition-field"
                  placeholder="e.g., event.data.amount"
                  value={conditionField}
                  onChange={(e) => setConditionField(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="condition-operator">Operator</Label>
                <Select
                  value={conditionOperator}
                  onValueChange={(value: any) => setConditionOperator(value)}
                >
                  <SelectTrigger id="condition-operator">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="eq">Equals</SelectItem>
                    <SelectItem value="neq">Not Equals</SelectItem>
                    <SelectItem value="gt">Greater Than</SelectItem>
                    <SelectItem value="gte">Greater Than or Equal</SelectItem>
                    <SelectItem value="lt">Less Than</SelectItem>
                    <SelectItem value="lte">Less Than or Equal</SelectItem>
                    <SelectItem value="in">In</SelectItem>
                    <SelectItem value="contains">Contains</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="condition-value">Value (JSON)</Label>
                <Input
                  id="condition-value"
                  placeholder="e.g., 1000"
                  value={conditionValue}
                  onChange={(e) => setConditionValue(e.target.value)}
                  required
                />
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="on-true">On True (Next Step ID)</Label>
                  <Input
                    id="on-true"
                    placeholder="e.g., send-notification"
                    value={onTrue}
                    onChange={(e) => setOnTrue(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="on-false">On False (Next Step ID)</Label>
                  <Input
                    id="on-false"
                    placeholder="e.g., auto-approve"
                    value={onFalse}
                    onChange={(e) => setOnFalse(e.target.value)}
                  />
                </div>
              </div>
            </div>
          )}

          {stepType === 'action' && (
            <div className="space-y-4 rounded-lg border p-4">
              <h4 className="font-medium">Action Configuration</h4>
              <div className="space-y-2">
                <Label htmlFor="action-type">Action Type</Label>
                <Select
                  value={actionType}
                  onValueChange={(value: any) => setActionType(value)}
                >
                  <SelectTrigger id="action-type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="allow">Allow</SelectItem>
                    <SelectItem value="block">Block</SelectItem>
                    <SelectItem value="execute">Execute</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="action-reason">Reason</Label>
                <Textarea
                  id="action-reason"
                  placeholder="Explain why this action is taken..."
                  value={actionReason}
                  onChange={(e) => setActionReason(e.target.value)}
                  rows={3}
                />
              </div>
            </div>
          )}

          {stepType === 'execute' && (
            <div className="space-y-4 rounded-lg border p-4">
              <h4 className="font-medium">Execute Configuration</h4>
              <div className="space-y-2">
                <Label htmlFor="execute-type">Execute Type</Label>
                <Select
                  value={executeType}
                  onValueChange={(value: any) => setExecuteType(value)}
                >
                  <SelectTrigger id="execute-type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="notify">Notify</SelectItem>
                    <SelectItem value="webhook">Webhook</SelectItem>
                    <SelectItem value="create_record">Create Record</SelectItem>
                    <SelectItem value="update_record">Update Record</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="execute-config">Configuration (JSON)</Label>
                <Textarea
                  id="execute-config"
                  placeholder='e.g., {"url": "https://api.example.com/notify"}'
                  value={executeConfig}
                  onChange={(e) => setExecuteConfig(e.target.value)}
                  rows={4}
                />
              </div>
            </div>
          )}

          {(stepType === 'parallel' || stepType === 'foreach') && (
            <div className="space-y-4 rounded-lg border p-4">
              <h4 className="font-medium">
                {stepType === 'parallel' ? 'Parallel' : 'For Each'} Configuration
              </h4>
              <p className="text-sm text-muted-foreground">
                Advanced configuration for {stepType} steps will be available in a future update.
              </p>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="next-step">Next Step ID</Label>
            <Input
              id="next-step"
              placeholder="e.g., final-step"
              value={nextStep}
              onChange={(e) => setNextStep(e.target.value)}
            />
          </div>

          <div className="flex justify-end gap-4">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit">Save Step</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

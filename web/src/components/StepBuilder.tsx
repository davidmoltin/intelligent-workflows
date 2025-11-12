import { useState } from 'react'
import type { Step } from '@/types/workflow'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Plus, Trash2, Edit, GripVertical } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { StepEditor } from './StepEditor'

interface StepBuilderProps {
  steps: Step[]
  onChange: (steps: Step[]) => void
}

export function StepBuilder({ steps, onChange }: StepBuilderProps) {
  const [editingStep, setEditingStep] = useState<number | null>(null)
  const [isCreating, setIsCreating] = useState(false)

  const handleAddStep = () => {
    setIsCreating(true)
  }

  const handleSaveNewStep = (step: Step) => {
    onChange([...steps, step])
    setIsCreating(false)
  }

  const handleUpdateStep = (index: number, step: Step) => {
    const newSteps = [...steps]
    newSteps[index] = step
    onChange(newSteps)
    setEditingStep(null)
  }

  const handleDeleteStep = (index: number) => {
    const newSteps = steps.filter((_, i) => i !== index)
    onChange(newSteps)
  }

  const handleMoveStep = (index: number, direction: 'up' | 'down') => {
    const newSteps = [...steps]
    const targetIndex = direction === 'up' ? index - 1 : index + 1

    if (targetIndex < 0 || targetIndex >= steps.length) return

    const temp = newSteps[index]
    newSteps[index] = newSteps[targetIndex]
    newSteps[targetIndex] = temp

    onChange(newSteps)
  }

  const getStepTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      condition: 'bg-blue-500',
      action: 'bg-green-500',
      execute: 'bg-purple-500',
      parallel: 'bg-orange-500',
      foreach: 'bg-pink-500',
    }
    return colors[type] || 'bg-gray-500'
  }

  return (
    <div className="space-y-4">
      {steps.length === 0 && !isCreating ? (
        <div className="flex flex-col items-center justify-center py-12 text-center border-2 border-dashed rounded-lg">
          <p className="text-muted-foreground mb-4">No steps defined yet</p>
          <Button onClick={handleAddStep}>
            <Plus className="mr-2 h-4 w-4" />
            Add First Step
          </Button>
        </div>
      ) : (
        <>
          {steps.map((step, index) => (
            <div key={index}>
              {editingStep === index ? (
                <StepEditor
                  step={step}
                  onSave={(updatedStep) => handleUpdateStep(index, updatedStep)}
                  onCancel={() => setEditingStep(null)}
                />
              ) : (
                <Card>
                  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                    <div className="flex items-center gap-3">
                      <div className="flex flex-col gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-4 w-4 p-0"
                          onClick={() => handleMoveStep(index, 'up')}
                          disabled={index === 0}
                        >
                          <GripVertical className="h-3 w-3" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-4 w-4 p-0"
                          onClick={() => handleMoveStep(index, 'down')}
                          disabled={index === steps.length - 1}
                        >
                          <GripVertical className="h-3 w-3" />
                        </Button>
                      </div>
                      <div className="flex items-center gap-2">
                        <Badge className={getStepTypeColor(step.type)}>
                          {step.type}
                        </Badge>
                        <CardTitle className="text-base">
                          {step.name || step.id}
                        </CardTitle>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setEditingStep(index)}
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteStep(index)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="text-sm text-muted-foreground space-y-1">
                      <div>ID: {step.id}</div>
                      {step.next && <div>Next: {step.next}</div>}
                      {step.type === 'condition' && step.condition && (
                        <div>
                          Condition: {step.condition.field} {step.condition.operator} {JSON.stringify(step.condition.value)}
                        </div>
                      )}
                      {step.type === 'action' && step.action && (
                        <div>Action: {step.action.type}</div>
                      )}
                      {step.type === 'execute' && step.execute && (
                        <div>Execute: {step.execute.length} action(s)</div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          ))}

          {isCreating && (
            <StepEditor
              step={null}
              onSave={handleSaveNewStep}
              onCancel={() => setIsCreating(false)}
            />
          )}

          {!isCreating && (
            <Button onClick={handleAddStep} variant="outline" className="w-full">
              <Plus className="mr-2 h-4 w-4" />
              Add Step
            </Button>
          )}
        </>
      )}
    </div>
  )
}

import { Handle, Position } from 'reactflow'
import { CheckCircle, XCircle, Zap } from 'lucide-react'
import type { Step } from '@/types/workflow'

interface ActionNodeProps {
  data: {
    label: string
    step?: Step
  }
  selected?: boolean
}

export function ActionNode({ data, selected }: ActionNodeProps) {
  const action = data.step?.action
  const actionType = action?.type || 'execute'

  const icons = {
    allow: CheckCircle,
    block: XCircle,
    execute: Zap,
    wait: Zap,
    require_approval: XCircle
  }

  const Icon = icons[actionType as keyof typeof icons] || Zap

  return (
    <div
      className={`px-4 py-3 shadow-lg rounded-lg bg-white border-2 min-w-[200px] ${
        selected ? 'border-green-600 ring-2 ring-green-300' : 'border-green-400'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-green-500 !border-2 !border-white !w-3 !h-3"
      />
      <div className="flex items-start gap-2">
        <div className="rounded-full bg-green-100 p-1.5 mt-0.5">
          <Icon className="h-4 w-4 text-green-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-xs font-semibold uppercase tracking-wide text-green-600">
            Action
          </div>
          <div className="font-medium text-gray-900 truncate">{data.label}</div>
          {action && (
            <div className="text-xs text-gray-600 mt-1">
              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                {action.type}
              </span>
              {action.reason && (
                <div className="mt-1 truncate text-gray-500">{action.reason}</div>
              )}
            </div>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-green-500 !border-2 !border-white !w-3 !h-3"
      />
    </div>
  )
}

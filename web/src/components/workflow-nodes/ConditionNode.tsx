import { Handle, Position } from 'reactflow'
import { GitBranch } from 'lucide-react'
import type { Step } from '@/types/workflow'

interface ConditionNodeProps {
  data: {
    label: string
    step?: Step
  }
  selected?: boolean
}

export function ConditionNode({ data, selected }: ConditionNodeProps) {
  const condition = data.step?.condition

  return (
    <div
      className={`px-4 py-3 shadow-lg rounded-lg bg-white border-2 min-w-[200px] ${
        selected ? 'border-blue-600 ring-2 ring-blue-300' : 'border-blue-400'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-blue-500 !border-2 !border-white !w-3 !h-3"
      />
      <div className="flex items-start gap-2">
        <div className="rounded-full bg-blue-100 p-1.5 mt-0.5">
          <GitBranch className="h-4 w-4 text-blue-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-xs font-semibold uppercase tracking-wide text-blue-600">
            Condition
          </div>
          <div className="font-medium text-gray-900 truncate">{data.label}</div>
          {condition && (
            <div className="text-xs text-gray-600 mt-1 space-y-0.5">
              {condition.field && (
                <div className="truncate">
                  <span className="font-medium">{condition.field}</span>
                  {condition.operator && ` ${condition.operator} `}
                  {condition.value !== undefined && (
                    <span className="font-mono">{String(condition.value)}</span>
                  )}
                </div>
              )}
              {(condition.and || condition.or) && (
                <div className="text-blue-500">Complex condition</div>
              )}
            </div>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        id="true"
        className="!bg-green-500 !border-2 !border-white !w-3 !h-3 !left-[30%]"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        id="false"
        className="!bg-red-500 !border-2 !border-white !w-3 !h-3 !left-[70%]"
      />
    </div>
  )
}

import { Handle, Position } from 'reactflow'
import { GitMerge } from 'lucide-react'
import type { Step } from '@/types/workflow'

interface ParallelNodeProps {
  data: {
    label: string
    step?: Step
  }
  selected?: boolean
}

export function ParallelNode({ data, selected }: ParallelNodeProps) {
  const nestedSteps = data.step?.steps || []

  return (
    <div
      className={`px-4 py-3 shadow-lg rounded-lg bg-white border-2 min-w-[200px] ${
        selected ? 'border-orange-600 ring-2 ring-orange-300' : 'border-orange-400'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-orange-500 !border-2 !border-white !w-3 !h-3"
      />
      <div className="flex items-start gap-2">
        <div className="rounded-full bg-orange-100 p-1.5 mt-0.5">
          <GitMerge className="h-4 w-4 text-orange-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-xs font-semibold uppercase tracking-wide text-orange-600">
            Parallel
          </div>
          <div className="font-medium text-gray-900 truncate">{data.label}</div>
          {nestedSteps.length > 0 && (
            <div className="text-xs text-gray-600 mt-1">
              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-100 text-orange-800">
                {nestedSteps.length} step{nestedSteps.length !== 1 ? 's' : ''}
              </span>
            </div>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-orange-500 !border-2 !border-white !w-3 !h-3"
      />
    </div>
  )
}

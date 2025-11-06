import { Handle, Position } from 'reactflow'
import { Repeat } from 'lucide-react'
import type { Step } from '@/types/workflow'

interface ForEachNodeProps {
  data: {
    label: string
    step?: Step
  }
  selected?: boolean
}

export function ForEachNode({ data, selected }: ForEachNodeProps) {
  const nestedSteps = data.step?.steps || []

  return (
    <div
      className={`px-4 py-3 shadow-lg rounded-lg bg-white border-2 min-w-[200px] ${
        selected ? 'border-pink-600 ring-2 ring-pink-300' : 'border-pink-400'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-pink-500 !border-2 !border-white !w-3 !h-3"
      />
      <div className="flex items-start gap-2">
        <div className="rounded-full bg-pink-100 p-1.5 mt-0.5">
          <Repeat className="h-4 w-4 text-pink-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-xs font-semibold uppercase tracking-wide text-pink-600">
            For Each
          </div>
          <div className="font-medium text-gray-900 truncate">{data.label}</div>
          {nestedSteps.length > 0 && (
            <div className="text-xs text-gray-600 mt-1">
              <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-pink-100 text-pink-800">
                {nestedSteps.length} step{nestedSteps.length !== 1 ? 's' : ''}
              </span>
            </div>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-pink-500 !border-2 !border-white !w-3 !h-3"
      />
    </div>
  )
}

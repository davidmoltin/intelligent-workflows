import { Handle, Position } from 'reactflow'
import { Code } from 'lucide-react'
import type { Step } from '@/types/workflow'

interface ExecuteNodeProps {
  data: {
    label: string
    step?: Step
  }
  selected?: boolean
}

export function ExecuteNode({ data, selected }: ExecuteNodeProps) {
  const execute = data.step?.execute

  return (
    <div
      className={`px-4 py-3 shadow-lg rounded-lg bg-white border-2 min-w-[200px] ${
        selected ? 'border-purple-600 ring-2 ring-purple-300' : 'border-purple-400'
      }`}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-purple-500 !border-2 !border-white !w-3 !h-3"
      />
      <div className="flex items-start gap-2">
        <div className="rounded-full bg-purple-100 p-1.5 mt-0.5">
          <Code className="h-4 w-4 text-purple-600" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="text-xs font-semibold uppercase tracking-wide text-purple-600">
            Execute
          </div>
          <div className="font-medium text-gray-900 truncate">{data.label}</div>
          {execute && execute.length > 0 && (
            <div className="text-xs text-gray-600 mt-1 space-y-1">
              {execute.map((action, idx) => (
                <div key={idx} className="flex items-center gap-1">
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">
                    {action.type}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-purple-500 !border-2 !border-white !w-3 !h-3"
      />
    </div>
  )
}

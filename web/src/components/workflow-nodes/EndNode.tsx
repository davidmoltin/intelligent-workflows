import { Handle, Position } from 'reactflow'
import { Flag } from 'lucide-react'

interface EndNodeProps {
  data: {
    label: string
  }
}

export function EndNode({ data }: EndNodeProps) {
  return (
    <div className="px-4 py-3 shadow-lg rounded-lg bg-gradient-to-r from-gray-500 to-gray-600 border-2 border-gray-700 min-w-[200px]">
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-gray-700 !border-2 !border-white !w-3 !h-3"
      />
      <div className="flex items-center gap-2 text-white">
        <Flag className="h-5 w-5" />
        <div className="flex-1">
          <div className="text-xs font-semibold uppercase tracking-wide opacity-90">
            End
          </div>
          <div className="font-medium">{data.label}</div>
        </div>
      </div>
    </div>
  )
}

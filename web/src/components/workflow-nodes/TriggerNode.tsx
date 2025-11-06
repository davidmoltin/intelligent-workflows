import { Handle, Position } from 'reactflow'
import { Play } from 'lucide-react'

interface TriggerNodeProps {
  data: {
    label: string
    trigger?: {
      type: string
      event?: string
      schedule?: string
    }
  }
}

export function TriggerNode({ data }: TriggerNodeProps) {
  return (
    <div className="px-4 py-3 shadow-lg rounded-lg bg-gradient-to-r from-purple-500 to-purple-600 border-2 border-purple-700 min-w-[200px]">
      <div className="flex items-center gap-2 text-white">
        <Play className="h-5 w-5" />
        <div className="flex-1">
          <div className="text-xs font-semibold uppercase tracking-wide opacity-90">
            Trigger
          </div>
          <div className="font-medium">{data.trigger?.type || 'Unknown'}</div>
          {data.trigger?.event && (
            <div className="text-xs opacity-90 mt-1">{data.trigger.event}</div>
          )}
          {data.trigger?.schedule && (
            <div className="text-xs opacity-90 mt-1">{data.trigger.schedule}</div>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-purple-700 !border-2 !border-white !w-3 !h-3"
      />
    </div>
  )
}

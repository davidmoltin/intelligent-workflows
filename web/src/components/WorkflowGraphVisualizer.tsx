import { useCallback, useMemo } from 'react'
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  ConnectionMode
} from 'reactflow'
import type { Node, NodeTypes } from 'reactflow'
import 'reactflow/dist/style.css'
import type { Trigger, Step } from '@/types/workflow'
import { workflowToGraph } from '@/lib/workflowGraph'
import {
  TriggerNode,
  ConditionNode,
  ActionNode,
  ExecuteNode,
  ParallelNode,
  ForEachNode,
  EndNode
} from './workflow-nodes'

interface WorkflowGraphVisualizerProps {
  trigger: Trigger
  steps: Step[]
  onNodeClick?: (step: Step) => void
  readonly?: boolean
}

const nodeTypes: NodeTypes = {
  trigger: TriggerNode,
  condition: ConditionNode,
  action: ActionNode,
  execute: ExecuteNode,
  parallel: ParallelNode,
  foreach: ForEachNode,
  end: EndNode
}

export function WorkflowGraphVisualizer({
  trigger,
  steps,
  onNodeClick,
  readonly = false
}: WorkflowGraphVisualizerProps) {
  // Convert workflow to graph format
  const { nodes: initialNodes, edges: initialEdges } = useMemo(() => {
    return workflowToGraph(trigger, steps)
  }, [trigger, steps])

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)

  // Update nodes and edges when workflow changes
  useMemo(() => {
    const { nodes: newNodes, edges: newEdges } = workflowToGraph(trigger, steps)
    setNodes(newNodes)
    setEdges(newEdges)
  }, [trigger, steps, setNodes, setEdges])

  // Handle node clicks
  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      if (readonly || !onNodeClick) return

      // Don't handle clicks on trigger or end nodes
      if (node.id === 'trigger' || node.id === 'end') return

      // Find the step and call the handler
      const step = steps.find(s => s.id === node.id)
      if (step) {
        onNodeClick(step)
      }
    },
    [steps, onNodeClick, readonly]
  )

  const minimapNodeColor = useCallback((node: Node) => {
    switch (node.type) {
      case 'trigger':
        return '#9333ea'
      case 'condition':
        return '#3b82f6'
      case 'action':
        return '#22c55e'
      case 'execute':
        return '#a855f7'
      case 'parallel':
        return '#f97316'
      case 'foreach':
        return '#ec4899'
      case 'end':
        return '#6b7280'
      default:
        return '#94a3b8'
    }
  }, [])

  return (
    <div className="w-full h-full bg-gray-50 rounded-lg border border-gray-200">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={handleNodeClick}
        nodeTypes={nodeTypes}
        connectionMode={ConnectionMode.Loose}
        fitView
        attributionPosition="bottom-right"
        nodesDraggable={!readonly}
        nodesConnectable={false}
        elementsSelectable={!readonly}
        minZoom={0.2}
        maxZoom={2}
      >
        <Background />
        <Controls />
        <MiniMap
          nodeColor={minimapNodeColor}
          maskColor="rgb(240, 240, 240, 0.6)"
          style={{
            backgroundColor: '#f9fafb'
          }}
        />
      </ReactFlow>
    </div>
  )
}

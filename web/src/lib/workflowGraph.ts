import type { Node, Edge } from 'reactflow'
import type { Step, Trigger } from '@/types/workflow'

export interface WorkflowNode extends Node {
  data: {
    step?: Step
    trigger?: Trigger
    label: string
    type: 'trigger' | 'condition' | 'action' | 'execute' | 'parallel' | 'foreach' | 'end'
  }
}

const LEVEL_HEIGHT = 150
const NODE_WIDTH = 250
const HORIZONTAL_SPACING = 100

/**
 * Calculate node positions using a hierarchical layout algorithm
 */
export function calculateLayout(nodes: WorkflowNode[], edges: Edge[]): WorkflowNode[] {
  // Build adjacency list
  const adjacency = new Map<string, string[]>()
  const inDegree = new Map<string, number>()

  nodes.forEach(node => {
    adjacency.set(node.id, [])
    inDegree.set(node.id, 0)
  })

  edges.forEach(edge => {
    const sources = adjacency.get(edge.source) || []
    sources.push(edge.target)
    adjacency.set(edge.source, sources)
    inDegree.set(edge.target, (inDegree.get(edge.target) || 0) + 1)
  })

  // Assign levels using BFS (topological sort)
  const levels = new Map<string, number>()
  const queue: string[] = []

  // Start with nodes that have no incoming edges
  nodes.forEach(node => {
    if ((inDegree.get(node.id) || 0) === 0) {
      queue.push(node.id)
      levels.set(node.id, 0)
    }
  })

  while (queue.length > 0) {
    const current = queue.shift()!
    const currentLevel = levels.get(current) || 0
    const neighbors = adjacency.get(current) || []

    neighbors.forEach(neighbor => {
      const neighborLevel = levels.get(neighbor) || 0
      levels.set(neighbor, Math.max(neighborLevel, currentLevel + 1))

      const degree = inDegree.get(neighbor) || 0
      inDegree.set(neighbor, degree - 1)

      if (inDegree.get(neighbor) === 0) {
        queue.push(neighbor)
      }
    })
  }

  // Group nodes by level
  const nodesByLevel = new Map<number, string[]>()
  levels.forEach((level, nodeId) => {
    const nodesAtLevel = nodesByLevel.get(level) || []
    nodesAtLevel.push(nodeId)
    nodesByLevel.set(level, nodesAtLevel)
  })

  // Assign positions
  const positionedNodes = nodes.map(node => {
    const level = levels.get(node.id) || 0
    const nodesAtLevel = nodesByLevel.get(level) || []
    const indexInLevel = nodesAtLevel.indexOf(node.id)
    const totalNodesAtLevel = nodesAtLevel.length

    // Center nodes horizontally at each level
    const totalWidth = (totalNodesAtLevel - 1) * (NODE_WIDTH + HORIZONTAL_SPACING)
    const startX = -totalWidth / 2

    return {
      ...node,
      position: {
        x: startX + indexInLevel * (NODE_WIDTH + HORIZONTAL_SPACING),
        y: level * LEVEL_HEIGHT
      }
    }
  })

  return positionedNodes
}

/**
 * Convert workflow steps to React Flow nodes
 */
export function workflowToNodes(
  trigger: Trigger,
  steps: Step[]
): WorkflowNode[] {
  const nodes: WorkflowNode[] = []

  // Add trigger node
  nodes.push({
    id: 'trigger',
    type: 'trigger',
    position: { x: 0, y: 0 },
    data: {
      trigger,
      label: `Trigger: ${trigger.type}`,
      type: 'trigger'
    }
  })

  // Add step nodes
  steps.forEach(step => {
    const label = step.name || step.id

    nodes.push({
      id: step.id,
      type: step.type,
      position: { x: 0, y: 0 }, // Will be calculated in layout
      data: {
        step,
        label,
        type: step.type
      }
    })

    // Handle nested steps (parallel/foreach)
    if (step.steps && step.steps.length > 0) {
      step.steps.forEach((nestedStep) => {
        const nestedLabel = nestedStep.name || nestedStep.id
        nodes.push({
          id: `${step.id}-nested-${nestedStep.id}`,
          type: nestedStep.type,
          position: { x: 0, y: 0 },
          data: {
            step: nestedStep,
            label: nestedLabel,
            type: nestedStep.type
          },
          parentNode: step.id
        })
      })
    }
  })

  // Add end node if there's at least one step
  if (steps.length > 0) {
    nodes.push({
      id: 'end',
      type: 'end',
      position: { x: 0, y: 0 },
      data: {
        label: 'End',
        type: 'end'
      }
    })
  }

  return nodes
}

/**
 * Convert workflow step routing to React Flow edges
 */
export function workflowToEdges(
  steps: Step[],
  hasTrigger: boolean = true
): Edge[] {
  const edges: Edge[] = []

  // Find the first step (the one not referenced by any other step)
  const referencedSteps = new Set<string>()
  steps.forEach(step => {
    if (step.next) referencedSteps.add(step.next)
    if (step.on_true) referencedSteps.add(step.on_true)
    if (step.on_false) referencedSteps.add(step.on_false)
  })

  const firstStep = steps.find(step => !referencedSteps.has(step.id))

  // Connect trigger to first step
  if (hasTrigger && firstStep) {
    edges.push({
      id: `trigger-${firstStep.id}`,
      source: 'trigger',
      target: firstStep.id,
      type: 'smoothstep'
    })
  }

  // Create edges based on step routing
  steps.forEach(step => {
    // Handle condition steps with true/false branches
    if (step.type === 'condition') {
      if (step.on_true) {
        edges.push({
          id: `${step.id}-true-${step.on_true}`,
          source: step.id,
          target: step.on_true,
          type: 'smoothstep',
          label: 'true',
          labelStyle: { fill: '#22c55e', fontWeight: 600 },
          style: { stroke: '#22c55e' }
        })
      }

      if (step.on_false) {
        edges.push({
          id: `${step.id}-false-${step.on_false}`,
          source: step.id,
          target: step.on_false,
          type: 'smoothstep',
          label: 'false',
          labelStyle: { fill: '#ef4444', fontWeight: 600 },
          style: { stroke: '#ef4444' }
        })
      }
    }

    // Handle regular next step connection
    if (step.next) {
      edges.push({
        id: `${step.id}-${step.next}`,
        source: step.id,
        target: step.next,
        type: 'smoothstep'
      })
    }

    // Handle nested steps (parallel/foreach)
    if (step.steps && step.steps.length > 0) {
      step.steps.forEach((nestedStep, index) => {
        // Connect parent to first nested step
        if (index === 0) {
          edges.push({
            id: `${step.id}-nested-${nestedStep.id}`,
            source: step.id,
            target: `${step.id}-nested-${nestedStep.id}`,
            type: 'smoothstep'
          })
        }

        // Connect nested steps sequentially
        if (index < step.steps!.length - 1) {
          const nextNested = step.steps![index + 1]
          edges.push({
            id: `${step.id}-nested-${nestedStep.id}-${nextNested.id}`,
            source: `${step.id}-nested-${nestedStep.id}`,
            target: `${step.id}-nested-${nextNested.id}`,
            type: 'smoothstep'
          })
        }
      })
    }
  })

  // Find steps with no outgoing edges (terminal steps)
  const stepsWithOutgoing = new Set<string>()
  edges.forEach(edge => {
    if (edge.source !== 'trigger' && !edge.source.includes('-nested-')) {
      stepsWithOutgoing.add(edge.source)
    }
  })

  // Connect terminal steps to end node
  steps.forEach(step => {
    if (!stepsWithOutgoing.has(step.id)) {
      edges.push({
        id: `${step.id}-end`,
        source: step.id,
        target: 'end',
        type: 'smoothstep'
      })
    }
  })

  return edges
}

/**
 * Convert workflow to positioned nodes and edges
 */
export function workflowToGraph(
  trigger: Trigger,
  steps: Step[]
): { nodes: WorkflowNode[]; edges: Edge[] } {
  const nodes = workflowToNodes(trigger, steps)
  const edges = workflowToEdges(steps, true)
  const positionedNodes = calculateLayout(nodes, edges)

  return { nodes: positionedNodes, edges }
}

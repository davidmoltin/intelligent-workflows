import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { workflowAPI, executionAPI, approvalAPI, eventAPI } from './client'
import type {
  CreateWorkflowRequest,
  UpdateWorkflowRequest,
  IngestEventRequest,
} from '@/types/workflow'

// Query Keys
export const queryKeys = {
  workflows: ['workflows'] as const,
  workflow: (id: string) => ['workflows', id] as const,
  executions: ['executions'] as const,
  execution: (id: string) => ['executions', id] as const,
  executionTrace: (id: string) => ['executions', id, 'trace'] as const,
  approvals: ['approvals'] as const,
  approval: (id: string) => ['approvals', id] as const,
}

// Workflow Hooks
export function useWorkflows(params?: {
  enabled?: boolean
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: [...queryKeys.workflows, params],
    queryFn: () => workflowAPI.list(params),
  })
}

export function useWorkflow(id: string) {
  return useQuery({
    queryKey: queryKeys.workflow(id),
    queryFn: () => workflowAPI.get(id),
    enabled: !!id,
  })
}

export function useCreateWorkflow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateWorkflowRequest) => workflowAPI.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workflows })
    },
  })
}

export function useUpdateWorkflow(id: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: UpdateWorkflowRequest) => workflowAPI.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workflow(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.workflows })
    },
  })
}

export function useDeleteWorkflow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => workflowAPI.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workflows })
    },
  })
}

export function useEnableWorkflow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => workflowAPI.enable(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workflow(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.workflows })
    },
  })
}

export function useDisableWorkflow() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => workflowAPI.disable(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workflow(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.workflows })
    },
  })
}

// Execution Hooks
export function useExecutions(params?: {
  workflow_id?: string
  status?: string
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: [...queryKeys.executions, params],
    queryFn: () => executionAPI.list(params),
  })
}

export function useExecution(id: string) {
  return useQuery({
    queryKey: queryKeys.execution(id),
    queryFn: () => executionAPI.get(id),
    enabled: !!id,
  })
}

export function useExecutionTrace(id: string) {
  return useQuery({
    queryKey: queryKeys.executionTrace(id),
    queryFn: () => executionAPI.getTrace(id),
    enabled: !!id,
  })
}

// Event Hooks
export function useIngestEvent() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: IngestEventRequest) => eventAPI.ingest(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.executions })
    },
  })
}

// Approval Hooks
export function useApprovals(params?: {
  status?: string
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: [...queryKeys.approvals, params],
    queryFn: () => approvalAPI.list(params),
  })
}

export function useApproval(id: string) {
  return useQuery({
    queryKey: queryKeys.approval(id),
    queryFn: () => approvalAPI.get(id),
    enabled: !!id,
  })
}

export function useApproveRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      approvalAPI.approve(id, reason),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.approval(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.approvals })
    },
  })
}

export function useRejectRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      approvalAPI.reject(id, reason),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.approval(id) })
      queryClient.invalidateQueries({ queryKey: queryKeys.approvals })
    },
  })
}

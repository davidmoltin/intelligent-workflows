import type {
  Workflow,
  CreateWorkflowRequest,
  UpdateWorkflowRequest,
  Execution,
  ApprovalRequest,
  IngestEventRequest,
} from '@/types/workflow'

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'

class APIError extends Error {
  status: number
  response?: any

  constructor(
    message: string,
    status: number,
    response?: any
  ) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.response = response
  }
}

async function fetchAPI<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`

  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new APIError(
        errorData.message || errorData.error || 'An error occurred',
        response.status,
        errorData
      )
    }

    if (response.status === 204) {
      return {} as T
    }

    return await response.json()
  } catch (error) {
    if (error instanceof APIError) {
      throw error
    }
    throw new APIError(
      error instanceof Error ? error.message : 'Network error',
      0
    )
  }
}

// Workflow API
export const workflowAPI = {
  list: async (params?: { enabled?: boolean; limit?: number; offset?: number }) => {
    const queryParams = new URLSearchParams()
    if (params?.enabled !== undefined) queryParams.set('enabled', String(params.enabled))
    if (params?.limit) queryParams.set('limit', String(params.limit))
    if (params?.offset) queryParams.set('offset', String(params.offset))

    const query = queryParams.toString()
    return fetchAPI<Workflow[]>(`/workflows${query ? `?${query}` : ''}`)
  },

  get: async (id: string) => {
    return fetchAPI<Workflow>(`/workflows/${id}`)
  },

  create: async (data: CreateWorkflowRequest) => {
    return fetchAPI<Workflow>('/workflows', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },

  update: async (id: string, data: UpdateWorkflowRequest) => {
    return fetchAPI<Workflow>(`/workflows/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },

  delete: async (id: string) => {
    return fetchAPI<void>(`/workflows/${id}`, {
      method: 'DELETE',
    })
  },

  enable: async (id: string) => {
    return fetchAPI<Workflow>(`/workflows/${id}/enable`, {
      method: 'POST',
    })
  },

  disable: async (id: string) => {
    return fetchAPI<Workflow>(`/workflows/${id}/disable`, {
      method: 'POST',
    })
  },
}

// Execution API
export const executionAPI = {
  list: async (params?: { workflow_id?: string; status?: string; limit?: number; offset?: number }) => {
    const queryParams = new URLSearchParams()
    if (params?.workflow_id) queryParams.set('workflow_id', params.workflow_id)
    if (params?.status) queryParams.set('status', params.status)
    if (params?.limit) queryParams.set('limit', String(params.limit))
    if (params?.offset) queryParams.set('offset', String(params.offset))

    const query = queryParams.toString()
    return fetchAPI<Execution[]>(`/executions${query ? `?${query}` : ''}`)
  },

  get: async (id: string) => {
    return fetchAPI<Execution>(`/executions/${id}`)
  },
}

// Event API
export const eventAPI = {
  ingest: async (data: IngestEventRequest) => {
    return fetchAPI<{ event_id: string; status: string }>('/events', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },
}

// Approval API
export const approvalAPI = {
  list: async (params?: { status?: string; limit?: number; offset?: number }) => {
    const queryParams = new URLSearchParams()
    if (params?.status) queryParams.set('status', params.status)
    if (params?.limit) queryParams.set('limit', String(params.limit))
    if (params?.offset) queryParams.set('offset', String(params.offset))

    const query = queryParams.toString()
    return fetchAPI<ApprovalRequest[]>(`/approvals${query ? `?${query}` : ''}`)
  },

  get: async (id: string) => {
    return fetchAPI<ApprovalRequest>(`/approvals/${id}`)
  },

  approve: async (id: string, reason: string) => {
    return fetchAPI<ApprovalRequest>(`/approvals/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
  },

  reject: async (id: string, reason: string) => {
    return fetchAPI<ApprovalRequest>(`/approvals/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    })
  },
}

// Health API
export const healthAPI = {
  health: async () => {
    return fetchAPI<{ status: string; version: string }>('/health')
  },

  ready: async () => {
    return fetchAPI<{
      status: string
      version: string
      checks: Record<string, string>
    }>('/ready')
  },
}

export { APIError }

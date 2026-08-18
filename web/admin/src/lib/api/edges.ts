import { apiFetch, toQuery } from './client'
import type {
  ComfyEdge,
  EdgeMetricsResponse,
  EdgePresence,
  EdgeStats,
  TaskRecord,
  EdgeHardware,
} from './types'

export function listEdges() {
  return apiFetch<ComfyEdge[]>('/api/v1/edges')
}

export function listPresence() {
  return apiFetch<EdgePresence[]>('/api/v1/edges/presence')
}

export function getEdge(id: string) {
  return apiFetch<ComfyEdge>(`/api/v1/edges/${encodeURIComponent(id)}`)
}

export function createEdge(body: {
  name: string
  description?: string
  enabled?: boolean
  capabilities?: string[]
}) {
  return apiFetch<ComfyEdge>('/api/v1/edges', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function patchEdge(
  id: string,
  body: {
    name?: string
    description?: string
    enabled?: boolean
    capabilities?: string[]
    refresh_hardware?: boolean
    hardware?: EdgeHardware
  }
) {
  return apiFetch<ComfyEdge>(`/api/v1/edges/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function rotateEdgeToken(id: string) {
  return apiFetch<ComfyEdge>(
    `/api/v1/edges/${encodeURIComponent(id)}/rotate-token`,
    { method: 'POST' }
  )
}

export function deleteEdge(id: string) {
  return apiFetch<void>(`/api/v1/edges/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function listEdgeTasks(id: string, params?: { limit?: number }) {
  return apiFetch<TaskRecord[]>(
    `/api/v1/edges/${encodeURIComponent(id)}/tasks${toQuery(params)}`
  )
}

export function getEdgeStats(id: string) {
  return apiFetch<EdgeStats>(`/api/v1/edges/${encodeURIComponent(id)}/stats`)
}

export function getEdgeMetrics(id: string, window: '1h' | '6h' | '24h' = '1h') {
  return apiFetch<EdgeMetricsResponse>(
    `/api/v1/edges/${encodeURIComponent(id)}/metrics?window=${window}`,
  )
}

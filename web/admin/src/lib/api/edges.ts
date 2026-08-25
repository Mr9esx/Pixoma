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
    subscribe_topics?: string[]
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

export type DeleteEdgeResult = {
  deleted: boolean
  failed_tasks?: number
}

export function deleteEdge(id: string, ack?: boolean) {
  return apiFetch<DeleteEdgeResult>(`/api/v1/edges/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    ...(ack ? { body: JSON.stringify({ ack_references: true }) } : {}),
  })
}

export function listEdgeTasks(
  id: string,
  params?: { status?: string; limit?: number; offset?: number }
) {
  return apiFetch<TaskRecord[]>(
    `/api/v1/edges/${encodeURIComponent(id)}/tasks${toQuery(params)}`
  )
}

export function getEdgeStats(id: string) {
  return apiFetch<EdgeStats>(`/api/v1/edges/${encodeURIComponent(id)}/stats`)
}

export type MetricsWindow = '1h' | '6h' | '24h'

export type MetricsRange =
  | { kind: 'preset'; window: MetricsWindow }
  | { kind: 'custom'; from: string; to: string }

export function getEdgeMetrics(
  id: string,
  window: MetricsWindow | 'custom' = '1h',
  range?: { from: string; to: string }
) {
  const params =
    window === 'custom' && range
      ? `?from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}`
      : `?window=${window}`
  return apiFetch<EdgeMetricsResponse>(
    `/api/v1/edges/${encodeURIComponent(id)}/metrics${params}`
  )
}

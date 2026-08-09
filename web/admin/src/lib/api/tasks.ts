import { apiFetch, toQuery } from './client'
import type { TaskRecord } from './types'

export function listTasks(params?: {
  status?: string
  q?: string
  limit?: number
  offset?: number
}) {
  return apiFetch<TaskRecord[]>(`/api/v1/tasks${toQuery(params)}`)
}

export function getTask(id: string) {
  return apiFetch<TaskRecord>(`/api/v1/tasks/${encodeURIComponent(id)}`)
}

export function cancelTask(id: string) {
  return apiFetch<TaskRecord>(
    `/api/v1/tasks/${encodeURIComponent(id)}/cancel`,
    { method: 'POST' },
  )
}

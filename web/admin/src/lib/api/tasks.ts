import { apiFetch, toQuery } from './client'
import type { TaskRecord } from './types'

export function listTasks(params?: {
  status?: string
  case_id?: number
  channel_id?: string
  dispatch_topic?: string
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

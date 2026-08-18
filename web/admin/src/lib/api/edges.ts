import { apiFetch, toQuery } from './client'
import type { ComfyInstance, TaskRecord } from './types'

export function listInstances() {
  return apiFetch<ComfyInstance[]>('/api/v1/comfy-instances')
}

export function getInstance(id: string) {
  return apiFetch<ComfyInstance>(
    `/api/v1/comfy-instances/${encodeURIComponent(id)}`,
  )
}

export function createInstance(body: {
  id: string
  base_url: string
  enabled?: boolean
  capabilities?: string[]
}) {
  return apiFetch<ComfyInstance>('/api/v1/comfy-instances', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function patchInstance(
  id: string,
  body: { base_url?: string; enabled?: boolean; capabilities?: string[] },
) {
  return apiFetch<ComfyInstance>(
    `/api/v1/comfy-instances/${encodeURIComponent(id)}`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
    },
  )
}

export function deleteInstance(id: string) {
  return apiFetch<void>(`/api/v1/comfy-instances/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function getInstanceSystem(id: string) {
  return apiFetch<unknown>(
    `/api/v1/comfy-instances/${encodeURIComponent(id)}/system`,
  )
}

export function getInstanceQueue(id: string) {
  return apiFetch<unknown>(
    `/api/v1/comfy-instances/${encodeURIComponent(id)}/queue`,
  )
}

export function listInstanceTasks(id: string, params?: { limit?: number }) {
  return apiFetch<TaskRecord[]>(
    `/api/v1/comfy-instances/${encodeURIComponent(id)}/tasks${toQuery(params)}`,
  )
}

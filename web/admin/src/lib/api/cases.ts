import { apiFetch, toQuery } from './client'
import type { CaseRecord } from './types'

export function listCases(params?: {
  q?: string
  enabled?: boolean
  limit?: number
  offset?: number
}) {
  return apiFetch<CaseRecord[]>(`/api/v1/cases${toQuery(params)}`)
}

export function getCase(id: number) {
  return apiFetch<CaseRecord>(`/api/v1/cases/${id}`)
}

export function createCase(body: Omit<CaseRecord, never>) {
  return apiFetch<CaseRecord>('/api/v1/cases', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function patchCase(id: number, body: Partial<CaseRecord>) {
  return apiFetch<CaseRecord>(`/api/v1/cases/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function enableCase(id: number) {
  return apiFetch<CaseRecord>(`/api/v1/cases/${id}/enable`, { method: 'POST' })
}

export function disableCase(id: number) {
  return apiFetch<CaseRecord>(`/api/v1/cases/${id}/disable`, { method: 'POST' })
}

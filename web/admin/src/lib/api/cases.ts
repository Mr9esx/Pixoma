import { apiFetch, toQuery } from './client'
import type { CaseRecord } from './types'

export function listCases(params?: {
  q?: string
  enabled?: boolean
  menu_key?: string
  limit?: number
  offset?: number
}) {
  return apiFetch<CaseRecord[]>(`/api/v1/cases${toQuery(params)}`)
}

export function getCase(id: string) {
  return apiFetch<CaseRecord>(`/api/v1/cases/${encodeURIComponent(id)}`)
}

export function createCase(body: Omit<CaseRecord, never>) {
  return apiFetch<CaseRecord>('/api/v1/cases', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function patchCase(id: string, body: Partial<CaseRecord>) {
  return apiFetch<CaseRecord>(`/api/v1/cases/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function enableCase(id: string) {
  return apiFetch<CaseRecord>(
    `/api/v1/cases/${encodeURIComponent(id)}/enable`,
    { method: 'POST' },
  )
}

export function disableCase(id: string) {
  return apiFetch<CaseRecord>(
    `/api/v1/cases/${encodeURIComponent(id)}/disable`,
    { method: 'POST' },
  )
}

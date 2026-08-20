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

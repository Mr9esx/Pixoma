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

export type DeleteCaseResult = {
  deleted: boolean
  removed_placements?: {
    channel_id: string
    channel_name?: string
    item_id: string
    label: string
    kind: string
  }[]
  failed_tasks?: number
  terminated_sessions?: number
}

export function deleteCase(id: number, ack?: boolean) {
  return apiFetch<DeleteCaseResult>(`/api/v1/cases/${id}`, {
    method: 'DELETE',
    ...(ack ? { body: JSON.stringify({ ack_references: true }) } : {}),
  })
}

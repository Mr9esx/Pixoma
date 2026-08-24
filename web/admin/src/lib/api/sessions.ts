import { apiFetch, toQuery } from './client'
import type { SessionRecord } from './types'

export function listSessions(params?: {
  user_id?: string
  case_id?: number
  status?: string
  q?: string
  limit?: number
  offset?: number
}) {
  return apiFetch<SessionRecord[]>(`/api/v1/sessions${toQuery(params)}`)
}

export function getSession(id: string) {
  return apiFetch<SessionRecord>(`/api/v1/sessions/${encodeURIComponent(id)}`)
}

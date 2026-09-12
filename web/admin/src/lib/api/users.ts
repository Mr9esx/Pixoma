import { apiFetch, toQuery } from './client'
import type { UserRecord } from './types'

export function listUsers(params?: {
  q?: string
  channel_id?: string
  external_user_id?: string
  limit?: number
  offset?: number
}) {
  return apiFetch<UserRecord[]>(`/api/v1/users${toQuery(params)}`)
}

export function getUser(id: string) {
  return apiFetch<UserRecord>(`/api/v1/users/${encodeURIComponent(id)}`)
}

export function updateUserAccess(id: string, access: UserRecord['access']) {
  return apiFetch<UserRecord>(
    `/api/v1/users/${encodeURIComponent(id)}/access`,
    {
      method: 'PUT',
      body: JSON.stringify({ access }),
    }
  )
}

export function getMCPToken(id: string) {
  return apiFetch<{ token?: string }>(
    `/api/v1/users/${encodeURIComponent(id)}/mcp-token`
  )
}

export function rotateMCPToken(id: string) {
  return apiFetch<{ token: string }>(
    `/api/v1/users/${encodeURIComponent(id)}/mcp-token/rotate`,
    { method: 'POST' }
  )
}

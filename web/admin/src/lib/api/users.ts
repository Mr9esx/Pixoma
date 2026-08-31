import { apiFetch, toQuery } from './client'
import type { UserRecord } from './types'

export function listUsers(params?: {
  q?: string
  tg_user_id?: number
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

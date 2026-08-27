import { apiFetch, toQuery } from './client'

export type AdminUser = {
  id: string
  username: string
  email: string
  nickname: string
  avatar_url: string
  role: 'admin' | 'operator' | 'viewer'
  enabled: boolean
  must_change_password: boolean
  last_login_at: string | null
  created_at: string
  updated_at: string
}

export function listAdminUsers(params?: { q?: string; limit?: number }) {
  return apiFetch<AdminUser[]>(`/api/v1/adminusers${toQuery(params)}`)
}

export function createAdminUser(input: {
  username: string
  email?: string
  nickname?: string
  password: string
}) {
  return apiFetch<AdminUser>('/api/v1/adminusers', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateAdminUser(
  id: string,
  input: {
    email?: string
    nickname?: string
    role?: 'admin' | 'operator' | 'viewer'
    enabled?: boolean
    password?: string
  }
) {
  return apiFetch<AdminUser>(`/api/v1/adminusers/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
}

export function deleteAdminUser(id: string) {
  return apiFetch<{ ok: boolean }>(
    `/api/v1/adminusers/${encodeURIComponent(id)}`,
    { method: 'DELETE' }
  )
}

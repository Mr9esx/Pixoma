import { apiFetch, setSessionToken } from './client'

export type SetupStatus = {
  initialized: boolean
  authenticated: boolean
  must_change_password: boolean
  username?: string
  wizard_step?: string
  restart_required?: boolean
}

export type SetupDraft = {
  placement: 'local' | 'remote'
  db_driver: string
  db_dsn: string
  blob_driver: string
  blob_root?: string
  blob_endpoint?: string
  blob_region?: string
  blob_bucket?: string
  blob_access_key?: string
  blob_secret_key?: string
  comfy_mock: boolean
  comfyui_base_url: string
  default_instance_id: string
  auto_spawn_edge: boolean
  telegram_bot_token?: string
}

export function fetchSetupStatus() {
  return apiFetch<SetupStatus>('/api/v1/setup/status')
}

export async function loginAdmin(username: string, password: string) {
  const res = await apiFetch<{
    ok: boolean
    token: string
    username: string
    must_change_password: boolean
    initialized: boolean
  }>('/api/v1/setup/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  })
  setSessionToken(res.token)
  return res
}

export async function logoutAdmin() {
  try {
    await apiFetch('/api/v1/setup/logout', { method: 'POST' })
  } finally {
    setSessionToken(null)
  }
}

export function changeAdminPassword(oldPassword: string, newPassword: string) {
  return apiFetch<{ ok: boolean }>('/api/v1/setup/password', {
    method: 'POST',
    body: JSON.stringify({
      old_password: oldPassword,
      new_password: newPassword,
    }),
  })
}

export function testDatabase(driver: string, dsn: string) {
  return apiFetch<{ ok: boolean; driver: string }>('/api/v1/setup/database', {
    method: 'POST',
    body: JSON.stringify({ driver, dsn }),
  })
}

export function saveSetupDraft(draft: SetupDraft) {
  return apiFetch<{ ok: boolean; placement: string }>('/api/v1/setup/draft', {
    method: 'POST',
    body: JSON.stringify(draft),
  })
}

export function finalizeSetup() {
  return apiFetch<{
    ok: boolean
    initialized: boolean
    restart_required: boolean
    message: string
  }>('/api/v1/setup/finalize', { method: 'POST' })
}

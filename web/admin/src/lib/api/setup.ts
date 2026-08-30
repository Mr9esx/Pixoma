import { apiFetch, setSessionToken } from './client'

export type CurrentUser = {
  username: string
  nickname: string
  role: 'admin' | 'operator' | 'viewer'
  email?: string
  avatar_url?: string
}

export function fetchCurrentUser() {
  return apiFetch<CurrentUser>('/api/v1/setup/me')
}

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
  proxy_kind?: string
  proxy_host?: string
  proxy_port?: number
  allow_self_registration?: boolean
}

export function fetchSetupStatus() {
  return apiFetch<SetupStatus>('/api/v1/setup/status')
}

export function fetchRegistrationStatus() {
  return apiFetch<{ enabled: boolean }>('/api/v1/auth/registration')
}

export async function registerAccount(input: {
  username: string
  email?: string
  nickname?: string
  password: string
}) {
  const res = await apiFetch<{
    ok: boolean
    token: string
    username: string
    role: string
  }>('/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify({
      username: input.username,
      email: input.email ?? '',
      nickname: input.nickname ?? '',
      password: input.password,
    }),
  })
  setSessionToken(res.token)
  return res
}

export async function loginAdmin(
  username: string,
  password: string,
  remember = false
) {
  const res = await apiFetch<{
    ok: boolean
    token: string
    username: string
    must_change_password: boolean
    initialized: boolean
  }>('/api/v1/setup/login', {
    method: 'POST',
    body: JSON.stringify({ username, password, remember }),
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

export function saveAdminProfile(input: {
  nickname: string
  email?: string
  avatarUrl?: string
}) {
  return apiFetch<{ ok: boolean }>('/api/v1/setup/profile', {
    method: 'POST',
    body: JSON.stringify({
      nickname: input.nickname,
      email: input.email ?? '',
      avatar_url: input.avatarUrl ?? '',
    }),
  })
}

export function changeAdminPassword(input: {
  newPassword: string
  oldPassword?: string
}) {
  return apiFetch<{ ok: boolean }>('/api/v1/setup/password', {
    method: 'POST',
    body: JSON.stringify({
      new_password: input.newPassword,
      ...(input.oldPassword !== undefined
        ? { old_password: input.oldPassword }
        : {}),
    }),
  })
}

export function testDatabase(driver: string, dsn: string) {
  return apiFetch<{ ok: boolean; driver: string }>('/api/v1/setup/database', {
    method: 'POST',
    body: JSON.stringify({ driver, dsn }),
  })
}

export function testBlob(input: {
  blob_driver: string
  blob_root?: string
  blob_endpoint?: string
  blob_region?: string
  blob_bucket?: string
  blob_access_key?: string
  blob_secret_key?: string
  auto_create_bucket?: boolean
}) {
  return apiFetch<{ ok: boolean; code?: string; bucket?: string }>(
    '/api/v1/setup/blob-test',
    {
      method: 'POST',
      body: JSON.stringify(input),
    }
  )
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
    restarting?: boolean
    message: string
  }>('/api/v1/setup/finalize', { method: 'POST' })
}

export function fetchPlatformSettings() {
  return apiFetch<{
    configured: boolean
    settings?: SetupDraft
    public_url?: string
  }>('/api/v1/setup/settings')
}

export function savePlatformSettings(draft: SetupDraft) {
  return apiFetch<{
    ok: boolean
    restart_required?: boolean
    restarting?: boolean
    message?: string
  }>('/api/v1/setup/settings', {
    method: 'PUT',
    body: JSON.stringify(draft),
  })
}

function isSetupReady(status: SetupStatus): boolean {
  return Boolean(status.initialized && !status.restart_required)
}

export async function waitForSetupReady(opts?: {
  timeoutMs?: number
  intervalMs?: number
}): Promise<SetupStatus> {
  const timeoutMs = opts?.timeoutMs ?? 60_000
  const intervalMs = opts?.intervalMs ?? 400
  const started = Date.now()
  for (;;) {
    try {
      const status = await fetchSetupStatus()
      if (isSetupReady(status)) return status
    } catch {
      // pixoma is down while it reloads
    }
    if (Date.now() - started > timeoutMs) {
      throw new Error('等待重启超时，看看跑 pixoma 的窗口有没有报错')
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs))
  }
}

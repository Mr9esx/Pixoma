export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

function baseURL(): string {
  const raw = import.meta.env.VITE_ADMIN_API_BASE as string | undefined
  if (!raw) {
    return ''
  }
  return raw.replace(/\/$/, '')
}

function sessionToken(): string {
  try {
    return sessionStorage.getItem('pixoma_admin_token') || ''
  } catch {
    return ''
  }
}

export function setSessionToken(token: string | null): void {
  try {
    if (token) sessionStorage.setItem('pixoma_admin_token', token)
    else sessionStorage.removeItem('pixoma_admin_token')
  } catch {
    // ignore
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${baseURL()}${path.startsWith('/') ? path : `/${path}`}`
  const token = sessionToken()
  let res: Response
  try {
    res = await fetch(url, {
      ...init,
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...init?.headers,
      },
    })
  } catch {
    throw new ApiError(
      0,
      '无法连接后台。请确认服务状态。',
    )
  }

  const text = await res.text()
  let body: unknown = undefined
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = text
    }
  }

  if (!res.ok) {
    const msg =
      typeof body === 'object' &&
      body !== null &&
      'error' in body &&
      typeof (body as { error: unknown }).error === 'string'
        ? (body as { error: string }).error
        : `Request failed (${res.status})`
    throw new ApiError(res.status, msg)
  }

  return body as T
}

export function toQuery(
  params?: Record<string, string | number | boolean | undefined | null>,
): string {
  if (!params) return ''
  const sp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null || v === '') continue
    sp.set(k, String(v))
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}

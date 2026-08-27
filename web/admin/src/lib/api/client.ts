export class ApiError extends Error {
  status: number
  code?: string
  constructor(status: number, message: string, code?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export function baseURL(): string {
  const raw = import.meta.env.VITE_ADMIN_API_BASE as string | undefined
  if (!raw) {
    return ''
  }
  return raw.replace(/\/$/, '')
}

export function sessionToken(): string {
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

// 这些接口由 Gate 放行、本身承担「未认证/密码错误」语义，401 不应触发全局登出。
const PUBLIC_AUTH_PATHS = [
  '/api/v1/setup/login',
  '/api/v1/setup/status',
  '/api/v1/auth/register',
  '/api/v1/auth/registration',
]

function redirectToLogin() {
  if (typeof window === 'undefined') return
  setSessionToken(null)
  const url = new URL('/login', window.location.origin)
  url.searchParams.set('expired', '1')
  window.location.assign(url.toString())
}

export async function apiFetch<T>(
  path: string,
  init?: RequestInit
): Promise<T> {
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
    throw new ApiError(0, '无法连接后台。请确认服务状态。')
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
    if (res.status === 401 && !PUBLIC_AUTH_PATHS.includes(path)) {
      redirectToLogin()
      throw new ApiError(res.status, '登录已失效，请重新登录')
    }
    const msg =
      typeof body === 'object' &&
      body !== null &&
      'error' in body &&
      typeof (body as { error: unknown }).error === 'string'
        ? (body as { error: string }).error
        : `Request failed (${res.status})`
    const code =
      typeof body === 'object' &&
      body !== null &&
      'code' in body &&
      typeof (body as { code: unknown }).code === 'string'
        ? (body as { code: string }).code
        : undefined
    throw new ApiError(res.status, msg, code)
  }

  return body as T
}

export function toQuery(
  params?: Record<string, string | number | boolean | undefined | null>
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

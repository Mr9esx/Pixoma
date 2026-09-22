export class ApiError extends Error {
  status: number
  /** 业务错误码，7 位整数。网络层失败时为 0。 */
  code: number
  /** 后端给出的原始技术原因，已脱敏，可直接展示。 */
  detail?: string
  constructor(status: number, message: string, code = 0, detail?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.detail = detail
  }
}

/** 后端所有 JSON 接口的统一响应封装。 */
type Envelope<T> = {
  message?: string
  code?: number
  data?: T
  error_detail?: string
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
  // 尽力清除服务端会话 cookie，避免 /api/v1/setup/status 仍判定已认证，
  // 导致登录页守卫把已登录用户重新弹回首页。
  fetch(`${baseURL()}/api/v1/setup/logout`, {
    method: 'POST',
    credentials: 'include',
  })
    .catch(() => {
      // 登出失败也继续跳转登录页
    })
    .finally(() => {
      window.location.assign(url.toString())
    })
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
        ...(init?.body && !(init.body instanceof FormData) ? { 'Content-Type': 'application/json' } : {}),
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

  const envelope: Envelope<T> =
    typeof body === 'object' && body !== null ? (body as Envelope<T>) : {}

  if (!res.ok) {
    if (res.status === 401 && !PUBLIC_AUTH_PATHS.includes(path)) {
      redirectToLogin()
      throw new ApiError(res.status, '登录已失效，请重新登录')
    }
    const message =
      typeof envelope.message === 'string' && envelope.message
        ? envelope.message
        : `请求失败（${res.status}）`
    const code = typeof envelope.code === 'number' ? envelope.code : 0
    const detail =
      typeof envelope.error_detail === 'string' ? envelope.error_detail : undefined
    throw new ApiError(res.status, message, code, detail)
  }

  // 非封装响应（二进制预览、SSE 等）原样返回。
  if (typeof envelope.code !== 'number' || !('data' in envelope)) {
    return body as T
  }
  return envelope.data as T
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

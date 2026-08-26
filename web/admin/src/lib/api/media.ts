import { ApiError, baseURL, sessionToken } from './client'

export type UploadedMedia = {
  key: string
  url: string
  mime: string
  size: number
}

/** 图片 / 视频白名单（与服务端一致）。 */
export const MEDIA_MIME_WHITELIST: Record<string, string> = {
  'image/png': '.png',
  'image/jpeg': '.jpg',
  'image/webp': '.webp',
  'image/gif': '.gif',
  'video/mp4': '.mp4',
  'video/webm': '.webm',
}

/** 默认单文件上限，与服务端一致。 */
export const MEDIA_MAX_BYTES = 25 * 1024 * 1024

function authHeaders(): Record<string, string> {
  const token = sessionToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

/** 上传单个媒体文件到管理端。 */
export async function uploadMedia(file: File): Promise<UploadedMedia> {
  const form = new FormData()
  form.append('file', file, file.name)
  let res: Response
  try {
    res = await fetch(`${baseURL()}/api/v1/media/`, {
      method: 'POST',
      body: form,
      credentials: 'include',
      headers: { Accept: 'application/json', ...authHeaders() },
    })
  } catch {
    throw new ApiError(0, '无法连接后台，上传失败')
  }
  let body: unknown = undefined
  try {
    body = await res.json()
  } catch {
    // ignore
  }
  if (!res.ok) {
    const msg =
      typeof body === 'object' &&
      body !== null &&
      'error' in body &&
      typeof (body as { error: unknown }).error === 'string'
        ? (body as { error: string }).error
        : `上传失败 (${res.status})`
    throw new ApiError(res.status, msg)
  }
  return body as UploadedMedia
}

/** 拼接媒体预览请求路径。key 形如 previews/<file>。 */
export function mediaApiPath(key: string): string {
  const trimmed = key.replace(/^\/+/, '')
  return `/api/v1/media/${trimmed}`
}

/** 拉取媒体为 Blob（携带鉴权），用于 <img>/<video> 预览。 */
export async function fetchMediaBlob(key: string): Promise<Blob> {
  let res: Response
  try {
    res = await fetch(`${baseURL()}/api/v1/media/${key.replace(/^\/+/, '')}`, {
      credentials: 'include',
      headers: authHeaders(),
    })
  } catch {
    throw new ApiError(0, '无法连接后台，加载媒体失败')
  }
  if (!res.ok) throw new ApiError(res.status, `加载媒体失败 (${res.status})`)
  return res.blob()
}

/** 由存储的 preview 值推导可预览的媒体 key；无法识别时返回 null。 */
export function resolveMediaKey(preview?: string): string | null {
  if (!preview) return null
  const p = preview.trim()
  if (!p) return null
  if (/^previews\/.+/i.test(p)) return p
  const m = p.match(/^\/?api\/v1\/media\/(previews\/.+)$/i)
  return m ? m[1] : null
}

/** 客户端预检：类型是否在白名单内。 */
export function isAllowedMedia(mime: string, filename: string): boolean {
  const ext = filename.split('.').pop()?.toLowerCase() ?? ''
  return (
    MEDIA_MIME_WHITELIST[mime.toLowerCase()] === `.${ext}` ||
    Object.entries(MEDIA_MIME_WHITELIST).some(
      ([m, e]) => m === mime.toLowerCase() || e === `.${ext}`
    )
  )
}

import { describe, it, expect, vi, afterEach } from 'vitest'
import { apiFetch, type ApiError } from './client'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.unstubAllEnvs()
})

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('apiFetch', () => {
  it('unwraps data from the success envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse({
          message: 'success',
          code: 2000000,
          data: [{ id: 'gpu-1' }],
        })
      )
    )
    const data = await apiFetch<{ id: string }[]>('/api/v1/edges')
    expect(data[0].id).toBe('gpu-1')
  })

  it('returns null data as-is', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse({ message: 'success', code: 2000000, data: null })
      )
    )
    await expect(apiFetch('/api/v1/cases/1', { method: 'DELETE' })).resolves.toBeNull()
  })

  it('passes through responses that are not enveloped', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response('data: {"sequence":1}\n\n', {
          status: 200,
          headers: { 'Content-Type': 'text/event-stream' },
        })
      )
    )
    await expect(apiFetch('/api/v1/studio/agui')).resolves.toBe(
      'data: {"sequence":1}\n\n'
    )
  })

  it('throws ApiError carrying message, code and error_detail', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse(
          {
            message: '创建用例失败。检查请求参数后重试。',
            code: 4000602,
            data: null,
            error_detail: 'name is required',
          },
          400
        )
      )
    )
    await expect(apiFetch('/api/v1/cases', { method: 'POST' })).rejects.toMatchObject({
      status: 400,
      message: '创建用例失败。检查请求参数后重试。',
      code: 4000602,
      detail: 'name is required',
    } satisfies Partial<ApiError>)
  })

  it('falls back to a generic message when the envelope has none', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ code: 5000005, data: null }, 500))
    )
    await expect(apiFetch('/api/v1/edges')).rejects.toMatchObject({
      status: 500,
      code: 5000005,
      message: '请求失败（500）',
    } satisfies Partial<ApiError>)
  })

  it('uses a relative /api path when VITE_ADMIN_API_BASE is empty', async () => {
    vi.stubEnv('VITE_ADMIN_API_BASE', '')
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse({ message: 'success', code: 2000000, data: { initialized: false } })
      )
    vi.stubGlobal('fetch', fetchMock)

    await apiFetch('/api/v1/setup/status')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/v1/setup/status',
      expect.objectContaining({
        credentials: 'include',
      })
    )
  })

  it('maps fetch failure to a connection error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new TypeError('Failed to fetch'))
    )
    await expect(apiFetch('/api/v1/setup/login')).rejects.toMatchObject({
      status: 0,
      message: '无法连接后台。请确认服务状态。',
    } satisfies Partial<ApiError>)
  })

  it('clears the server session and redirects on a protected 401', async () => {
    const assign = vi.fn()
    vi.stubGlobal('window', {
      location: { origin: 'http://x', assign },
    })
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ message: '登录已失效。重新登录。', code: 4010107, data: null }, 401)
      )
      .mockResolvedValue(new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(apiFetch('/api/v1/adminusers')).rejects.toMatchObject({
      status: 401,
    })

    const logoutCall = fetchMock.mock.calls.find(([url]) =>
      String(url).endsWith('/api/v1/setup/logout')
    )
    expect(logoutCall).toBeDefined()
    expect(logoutCall?.[1]).toMatchObject({
      method: 'POST',
      credentials: 'include',
    })
    await vi.waitFor(() =>
      expect(assign).toHaveBeenCalledWith('http://x/login?expired=1')
    )
  })
})

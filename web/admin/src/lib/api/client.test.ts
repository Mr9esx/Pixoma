import { describe, it, expect, vi, afterEach } from 'vitest'
import { apiFetch, type ApiError } from './client'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.unstubAllEnvs()
})

describe('apiFetch', () => {
  it('throws ApiError with backend error message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: 'case not found' }), {
          status: 404,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    )
    await expect(apiFetch('/api/v1/cases/missing')).rejects.toMatchObject({
      status: 404,
      message: 'case not found',
    } satisfies Partial<ApiError>)
  })

  it('returns parsed JSON on 2xx', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([{ id: 'gpu-1' }]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    )
    const data = await apiFetch<{ id: string }[]>('/api/v1/edges')
    expect(data[0].id).toBe('gpu-1')
  })

  it('uses a relative /api path when VITE_ADMIN_API_BASE is empty', async () => {
    vi.stubEnv('VITE_ADMIN_API_BASE', '')
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ initialized: false }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
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
})

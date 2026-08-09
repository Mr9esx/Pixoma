import { describe, it, expect, vi, afterEach } from 'vitest'
import { listInstances, createInstance } from './instances'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('instances API', () => {
  it('listInstances GETs /api/v1/comfy-instances', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 'gpu-1' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listInstances()

    expect(data[0].id).toBe('gpu-1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/comfy-instances',
      expect.objectContaining({
        headers: expect.objectContaining({ Accept: 'application/json' }),
      }),
    )
    const init = fetchMock.mock.calls[0][1] as RequestInit | undefined
    expect(init?.method).toBeUndefined()
  })

  it('createInstance POSTs JSON body to /api/v1/comfy-instances', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: 'gpu-2',
          base_url: 'http://localhost:8188',
          enabled: true,
          capabilities: ['img'],
        }),
        {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    const body = {
      id: 'gpu-2',
      base_url: 'http://localhost:8188',
      enabled: true,
      capabilities: ['img'],
    }
    const data = await createInstance(body)

    expect(data.id).toBe('gpu-2')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/comfy-instances',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(body),
      }),
    )
  })
})

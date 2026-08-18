import { describe, it, expect, vi, afterEach } from 'vitest'
import { listEdges, createEdge, listPresence } from './edges'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('edges API', () => {
  it('listEdges GETs /api/v1/edges', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 'gpu-1' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listEdges()

    expect(data[0].id).toBe('gpu-1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/edges',
      expect.objectContaining({
        headers: expect.objectContaining({ Accept: 'application/json' }),
      }),
    )
    const init = fetchMock.mock.calls[0][1] as RequestInit | undefined
    expect(init?.method).toBeUndefined()
  })

  it('listPresence GETs /api/v1/edges/presence', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify([
          { id: 'gpu-1', edge_online: true, comfy_running: false },
        ]),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listPresence()

    expect(data[0].edge_online).toBe(true)
    expect(data[0].comfy_running).toBe(false)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/edges/presence',
      expect.objectContaining({
        headers: expect.objectContaining({ Accept: 'application/json' }),
      }),
    )
  })

  it('createEdge POSTs JSON body to /api/v1/edges', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: 'gpu-2',
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
      name: 'gpu-2',
      description: 'night jobs',
      enabled: true,
      capabilities: ['img'],
    }
    const data = await createEdge(body)

    expect(data.id).toBe('gpu-2')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/edges',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(body),
      }),
    )
  })
})

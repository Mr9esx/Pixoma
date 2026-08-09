import { describe, it, expect, vi, afterEach } from 'vitest'
import { listSessions, getSession } from './sessions'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('sessions API', () => {
  it('listSessions GETs /api/v1/sessions with filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 's1', user_id: 'u1' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listSessions({ user_id: 'u1', status: 'active' })

    expect(data[0].id).toBe('s1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/sessions?user_id=u1&status=active',
      expect.anything(),
    )
  })

  it('getSession GETs /api/v1/sessions/{id}', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 's1', status: 'active' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await getSession('s1')

    expect(data.id).toBe('s1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/sessions/s1',
      expect.anything(),
    )
  })
})

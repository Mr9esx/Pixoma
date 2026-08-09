import { describe, it, expect, vi, afterEach } from 'vitest'
import { listUsers, getUser } from './users'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('users API', () => {
  it('listUsers GETs /api/v1/users with filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 'u1', tg_user_id: 42 }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listUsers({ tg_user_id: 42, q: 'bob' })

    expect(data[0].tg_user_id).toBe(42)
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/users?tg_user_id=42&q=bob',
      expect.anything(),
    )
  })

  it('getUser GETs /api/v1/users/{id}', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'u1', username: 'bob' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await getUser('u1')

    expect(data.id).toBe('u1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/users/u1',
      expect.anything(),
    )
  })
})

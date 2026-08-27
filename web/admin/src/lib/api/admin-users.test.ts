import { describe, it, expect, vi, afterEach } from 'vitest'
import {
  listAdminUsers,
  createAdminUser,
  updateAdminUser,
  deleteAdminUser,
} from './admin-users'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('admin-users API', () => {
  it('listAdminUsers GETs /api/v1/adminusers with filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify([{ id: 'u1', username: 'bob' }]), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await listAdminUsers({ q: 'bob' })

    expect(data[0].id).toBe('u1')
    expect(fetchMock).toHaveBeenCalledWith(
      'http://127.0.0.1:8081/api/v1/adminusers?q=bob',
      expect.anything(),
    )
  })

  it('createAdminUser POSTs the payload', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: 'u2', username: 'alice' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const data = await createAdminUser({
      username: 'alice',
      email: 'a@b.com',
      password: 'secret-1',
    })

    expect(data.id).toBe('u2')
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:8081/api/v1/adminusers')
    expect(init.method).toBe('POST')
    expect(JSON.parse(init.body)).toMatchObject({
      username: 'alice',
      email: 'a@b.com',
    })
  })

  it('updateAdminUser PATCHes the user', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    await updateAdminUser('u1', { enabled: false, password: 'secret-1' })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:8081/api/v1/adminusers/u1')
    expect(init.method).toBe('PATCH')
    expect(JSON.parse(init.body)).toMatchObject({ enabled: false })
  })

  it('deleteAdminUser DELETEs the user', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(JSON.stringify({ ok: true }), {
        status: 200,
      }))
    vi.stubGlobal('fetch', fetchMock)

    await deleteAdminUser('u1')

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://127.0.0.1:8081/api/v1/adminusers/u1')
    expect(init.method).toBe('DELETE')
  })
})

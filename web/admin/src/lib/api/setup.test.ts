import { describe, it, expect, vi, afterEach } from 'vitest'
import { fetchSetupStatus, loginAdmin } from './setup'
import { setSessionToken } from './client'

afterEach(() => {
  vi.unstubAllGlobals()
  setSessionToken(null)
})

describe('setup api', () => {
  it('reads setup status', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            initialized: false,
            authenticated: false,
            must_change_password: true,
            live_demo: true,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    )
    const s = await fetchSetupStatus()
    expect(s.initialized).toBe(false)
    expect(s.live_demo).toBe(true)
  })

  it('stores session token after login', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            ok: true,
            token: 'sess-1',
            username: 'admin',
            must_change_password: true,
            initialized: false,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    )
    await loginAdmin('admin', 'x')
    expect(sessionStorage.getItem('pixoma_admin_token')).toBe('sess-1')
  })
})

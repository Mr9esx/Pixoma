import { describe, it, expect } from 'vitest'
import { nextAdminPath } from './setup-guard'

describe('nextAdminPath', () => {
  it('sends uninitialized guests to login first', () => {
    expect(
      nextAdminPath(
        { initialized: false, authenticated: false, must_change_password: true },
        '/',
      ),
    ).toBe('/login')
  })

  it('sends logged-in uninitialized users to setup', () => {
    expect(
      nextAdminPath(
        { initialized: false, authenticated: true, must_change_password: true },
        '/',
      ),
    ).toBe('/setup')
  })

  it('sends initialized guests to login', () => {
    expect(
      nextAdminPath(
        { initialized: true, authenticated: false, must_change_password: false },
        '/',
      ),
    ).toBe('/login')
  })

  it('keeps users on setup until restart', () => {
    expect(
      nextAdminPath(
        {
          initialized: true,
          authenticated: true,
          must_change_password: false,
          restart_required: true,
        },
        '/',
      ),
    ).toBe('/setup')
  })

  it('lets authenticated users into the shell', () => {
    expect(
      nextAdminPath(
        { initialized: true, authenticated: true, must_change_password: false },
        '/',
      ),
    ).toBeNull()
  })
})

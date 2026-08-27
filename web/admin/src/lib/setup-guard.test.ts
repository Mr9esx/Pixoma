import { describe, it, expect } from 'vitest'
import { nextAdminPath, nextRegistrationPath } from './setup-guard'

describe('nextAdminPath', () => {
  it('sends uninitialized guests to login first', () => {
    expect(
      nextAdminPath(
        {
          initialized: false,
          authenticated: false,
          must_change_password: true,
        },
        '/'
      )
    ).toBe('/login')
  })

  it('sends logged-in uninitialized users to setup', () => {
    expect(
      nextAdminPath(
        { initialized: false, authenticated: true, must_change_password: true },
        '/'
      )
    ).toBe('/setup')
  })

  it('sends initialized guests to login', () => {
    expect(
      nextAdminPath(
        {
          initialized: true,
          authenticated: false,
          must_change_password: false,
        },
        '/'
      )
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
        '/'
      )
    ).toBe('/setup')
  })

  it('lets authenticated users into the shell', () => {
    expect(
      nextAdminPath(
        { initialized: true, authenticated: true, must_change_password: false },
        '/'
      )
    ).toBeNull()
  })

  it('keeps the wizard up while pixoma is reloading', () => {
    expect(
      nextAdminPath(
        {
          initialized: true,
          authenticated: true,
          must_change_password: false,
          restart_required: true,
        },
        '/setup'
      )
    ).toBeNull()
  })
})


describe('nextRegistrationPath', () => {
  it('allows registration when the platform is initialized and open', () => {
    expect(
      nextRegistrationPath(
        { initialized: true, authenticated: false, must_change_password: false },
        true,
        '/register'
      )
    ).toBeNull()
  })

  it('redirects to login when registration is closed', () => {
    expect(
      nextRegistrationPath(
        { initialized: true, authenticated: false, must_change_password: false },
        false,
        '/register'
      )
    ).toBe('/login')
  })

  it('sends uninitialized guests away from registration', () => {
    expect(
      nextRegistrationPath(
        { initialized: false, authenticated: false, must_change_password: true },
        true,
        '/register'
      )
    ).toBe('/login')
  })

  it('sends authenticated users into the shell', () => {
    expect(
      nextRegistrationPath(
        { initialized: true, authenticated: true, must_change_password: false },
        true,
        '/register'
      )
    ).toBe('/')
  })
})

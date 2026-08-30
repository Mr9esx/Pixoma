import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../..')

function read(rel: string) {
  return readFileSync(join(root, rel), 'utf8')
}

describe('self-registration page', () => {
  it('is gated behind an initialized platform with open registration', () => {
    const route = read('src/routes/register.tsx')
    expect(route).toContain("createFileRoute('/register'")
    expect(route).toMatch(/fetchSetupStatus/)
    expect(route).toMatch(/fetchRegistrationStatus/)
    expect(route).toMatch(/nextRegistrationPath/)
  })

  it('collects username, email, nickname, and password for registration', () => {
    const form = read('src/features/auth/register-form.tsx')
    expect(form).toMatch(/htmlFor='register-username'/)
    expect(form).toMatch(/htmlFor='register-email'/)
    expect(form).toMatch(/htmlFor='register-nickname'/)
    expect(form).toMatch(/htmlFor='register-password'/)
    expect(form).toMatch(/registerAccount/)
  })

  it('submits to the register endpoint and signs in on success', () => {
    const api = read('src/lib/api/setup.ts')
    expect(api).toMatch(/registerAccount/)
    expect(api).toMatch(/\/api\/v1\/auth\/register/)
    expect(api).toMatch(/setSessionToken/)
  })

  it('links back to login', () => {
    expect(read('src/features/auth/register-form.tsx')).toMatch(/to='\/login'/)
  })

  it('registers the route in the generated tree', () => {
    const tree = read('src/routeTree.gen.ts')
    expect(tree).toContain("'/register'")
  })
})

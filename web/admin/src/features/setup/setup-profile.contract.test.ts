import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../..')

function read(rel: string) {
  return readFileSync(join(root, rel), 'utf8')
}

describe('setup wizard admin profile step', () => {
  it('inserts the「如何称呼您」step right after password', () => {
    const steps = read('src/features/setup/setup-steps.ts')
    expect(steps).toMatch(/title: '如何称呼您'/)
    const idx = steps.indexOf("'profile'")
    const pwdIdx = steps.indexOf("'password'")
    expect(idx).toBeGreaterThan(pwdIdx)
  })

  it('advances from password to profile before configuring the database', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/await changeAdminPassword/ )
    expect(wizard).toMatch(/setStep\('profile'\)/)
    expect(wizard).toMatch(/step === 'profile'/)
  })

  it('collects nickname, email, and avatar and lets the admin skip', () => {
    const wizard = read('src/features/setup/setup-wizard.tsx')
    expect(wizard).toMatch(/htmlFor='profile-nickname'/)
    expect(wizard).toMatch(/htmlFor='profile-email'/)
    expect(wizard).toMatch(/htmlFor='profile-avatar'/)
    expect(wizard).toMatch(/saveAdminProfile/)
  })

  it('sends the profile to the setup backend', () => {
    const api = read('src/lib/api/setup.ts')
    expect(api).toMatch(/saveAdminProfile/)
    expect(api).toMatch(/\/api\/v1\/setup\/profile/)
    expect(api).toMatch(/avatar_url/)
  })
})

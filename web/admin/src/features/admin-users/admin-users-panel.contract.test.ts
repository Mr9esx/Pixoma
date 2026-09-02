import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('admin users panel', () => {
  it('exists as a settings feature block', () => {
    expect(existsSync(join(here, 'admin-users-panel.tsx'))).toBe(true)
    expect(existsSync(join(here, '../../lib/api/admin-users.ts'))).toBe(true)
  })

  it('lists console users with a readable table and search', () => {
    const panel = read('admin-users-panel.tsx')
    expect(panel).toMatch(/listAdminUsers/)
    expect(panel).toMatch(/admin-users-search/)
    expect(panel).toMatch(/admin-users-create/)
  })

  it('offers add, enable/disable, reset password, and delete entry points', () => {
    const panel = read('admin-users-panel.tsx')
    expect(panel).toMatch(/createAdminUser/)
    expect(panel).toMatch(/updateAdminUser/)
    expect(panel).toMatch(/deleteAdminUser/)
    expect(panel).toMatch(/admin-user-password/)
    expect(panel).toMatch(/admin-reset-password/)
    expect(panel).toMatch(/ConfirmDialog/)
  })

  it('keeps password hashes out of the frontend model', () => {
    const api = read('../../lib/api/admin-users.ts')
    expect(api).not.toMatch(/password_hash/i)
    expect(api).not.toMatch(/passwordHash/i)
  })
})

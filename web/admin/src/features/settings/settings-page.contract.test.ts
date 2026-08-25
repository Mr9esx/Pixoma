import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('settings page', () => {
  it('exists under the app shell', () => {
    expect(existsSync(join(here, 'settings-page.tsx'))).toBe(true)
    expect(
      existsSync(join(here, '../../routes/_app/settings/index.tsx')),
    ).toBe(true)
  })

  it('lets operators change password, storage, and network after setup', () => {
    const page = read('settings-page.tsx')
    expect(page).toMatch(/htmlFor=['"]old-password['"]/)
    expect(page).toMatch(/htmlFor=['"]confirm-password['"]/)
    expect(page).toMatch(/htmlFor=['"]blob-driver['"]/)
    expect(page).toMatch(/htmlFor=['"]blob-root['"]/)
    expect(page).not.toMatch(/htmlFor=['"]tg-token['"]/)
    expect(page).toMatch(/htmlFor=['"]proxy-kind['"]/)
    expect(page).toMatch(/htmlFor=['"]proxy-host['"]/)
    expect(page).toMatch(/htmlFor=['"]proxy-port['"]/)
    expect(page).toMatch(/tabNetwork/)
    expect(page).toMatch(/savePlatformSettings/)
    expect(page).toMatch(/changeAdminPassword/)
    expect(page).toMatch(/waitForSetupReady/)
    expect(page).not.toMatch(/Comfy Mock/)
    expect(page).not.toMatch(/htmlFor=['"]comfy-url['"]/)
    expect(page).toMatch(/TabsList/)
    expect(page).not.toMatch(/RadioGroup/)
    expect(page).not.toMatch(/\/instances/)
  })

  it('shows the business database driver and DSN read-only', () => {
    const page = read('settings-page.tsx')
    expect(page).toMatch(/initial\.db_driver/)
    expect(page).toMatch(/initial\.db_dsn/)
    expect(page).not.toMatch(/htmlFor=['"]db-driver['"]/)
    expect(page).not.toMatch(/htmlFor=['"]db-dsn['"]/)
    expect(page).not.toMatch(/testDatabase\(/)
    expect(page).not.toMatch(/DB_DSN_PLACEHOLDER/)
  })

  it('settings deep-links to a tab via ?tab=', () => {
    const route = read('../../routes/_app/settings/index.tsx')
    const page = read('settings-page.tsx')
    expect(route).toMatch(/validateSearch/)
    expect(route).toMatch(/search\.tab/)
    expect(page).toMatch(/initialTab/)
  })
})

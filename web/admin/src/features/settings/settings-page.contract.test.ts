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
})

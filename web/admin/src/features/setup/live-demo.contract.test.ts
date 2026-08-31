import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = join(dirname(fileURLToPath(import.meta.url)), '../../..')

function read(rel: string) {
  return readFileSync(join(root, rel), 'utf8')
}

describe('readonly live demo UX', () => {
  it('shows fixed credentials on the login page', () => {
    const source = read('src/features/setup/login-page.tsx')
    expect(source).toMatch(/showLiveDemoHint = status\.live_demo === true/)
    expect(source).toMatch(/data-testid='live-demo-credentials'/)
    expect(source).toMatch(/admin \/ 123456/)
    expect(source).toMatch(/Live Demo 为只读模式/)
  })

  it('propagates live demo status through login routing', () => {
    expect(read('src/routes/login.tsx')).toMatch(/statusLiveDemo: status\.live_demo/)
  })

  it('filters settings navigation in live demo mode', () => {
    const source = read('src/components/layout/app-sidebar.tsx')
    expect(source).toMatch(/filterMenuGroupsForDemo/)
    expect(source).toMatch(/isLiveDemo/)
  })

  it('redirects direct settings visits in live demo mode', () => {
    const source = read('src/routes/_app/settings/index.tsx')
    expect(source).toMatch(/beforeLoad/)
    expect(source).toMatch(/status\.live_demo/)
    expect(source).toMatch(/redirect\(\{ to: '\/' \}\)/)
  })
})

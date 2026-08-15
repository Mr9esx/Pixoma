import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const src = join(dirname(fileURLToPath(import.meta.url)))
const root = join(src, '..')

function read(rel: string) {
  return readFileSync(join(root, rel), 'utf8')
}

describe('admin auth gates removed', () => {
  it('does not depend on @clerk packages', () => {
    const pkg = JSON.parse(read('package.json')) as {
      dependencies?: Record<string, string>
      pnpm?: { onlyBuiltDependencies?: string[] }
    }

    const depNames = Object.keys(pkg.dependencies ?? {})
    expect(depNames.some((name) => name.startsWith('@clerk/'))).toBe(false)
    expect(pkg.pnpm?.onlyBuiltDependencies ?? []).not.toContain('@clerk/shared')
  })

  it('main entry has no ClerkProvider and no sign-in redirect gate', () => {
    const main = read('src/main.tsx')

    expect(main).not.toMatch(/ClerkProvider/)
    expect(main).not.toMatch(/@clerk\//)
    expect(main).not.toMatch(/useAuthStore/)
    expect(main).not.toMatch(/['"]\/sign-in['"]/)
  })

  it('uses unauthenticated _app layout and drops auth/clerk route trees', () => {
    expect(existsSync(join(src, 'routes/_app.tsx'))).toBe(true)
    expect(existsSync(join(src, 'routes/_app/index.tsx'))).toBe(true)
    expect(existsSync(join(src, 'routes/_authenticated'))).toBe(false)
    expect(existsSync(join(src, 'routes/(auth)'))).toBe(false)
    expect(existsSync(join(src, 'routes/clerk'))).toBe(false)

    const appLayout = read('src/routes/_app.tsx')
    expect(appLayout).toMatch(/createFileRoute\(['"]\/_app['"]\)/)
    expect(appLayout).toMatch(/fetchSetupStatus/)
    expect(appLayout).not.toMatch(/ClerkProvider|SignedIn|redirect.*sign-in/)

    const index = read('src/routes/_app/index.tsx')
    expect(index).toMatch(/createFileRoute\(['"]\/_app\/['"]\)/)
  })

  it('generated route tree serves / under _app without clerk or auth paths', () => {
    const tree = read('src/routeTree.gen.ts')

    expect(tree).toMatch(/\/_app/)
    expect(tree).not.toMatch(/clerk/)
    expect(tree).not.toMatch(/\(auth\)/)
    expect(tree).not.toMatch(/_authenticated/)
    expect(tree).not.toMatch(/\/sign-in/)
  })
})

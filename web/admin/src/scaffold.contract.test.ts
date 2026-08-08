import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

describe('admin scaffold package.json', () => {
  it('uses pnpm and exposes dev/build/test scripts', () => {
    const pkg = JSON.parse(readFileSync(join(root, 'package.json'), 'utf8')) as {
      packageManager?: string
      scripts?: Record<string, string>
    }

    expect(pkg.packageManager?.startsWith('pnpm@')).toBe(true)
    expect(pkg.scripts?.dev).toBeTypeOf('string')
    expect(pkg.scripts?.build).toBeTypeOf('string')
    expect(pkg.scripts?.test).toBeTypeOf('string')
  })
})

import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const adminRoot = join(dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = join(adminRoot, '../..')

describe('make dev loop', () => {
  it('Makefile dev target runs scripts/dev.sh', () => {
    const mk = readFileSync(join(repoRoot, 'Makefile'), 'utf8')
    expect(mk).toMatch(/^\.PHONY:.*\bdev\b/m)
    expect(mk).toMatch(/^dev:\n\tbash scripts\/dev\.sh/m)
    expect(mk).toContain('PORT ?= 30808')
    expect(mk).toContain('HTTP_ADDR ?= 0.0.0.0:$(PORT)')
    expect(mk).toMatch(/export PORT HTTP_ADDR/)
  })

  it('dev.sh starts pixoma and vite, prints both URLs, and traps signals', () => {
    const sh = readFileSync(join(repoRoot, 'scripts/dev.sh'), 'utf8')
    expect(sh).toContain('管理页面: http://127.0.0.1:5173')
    expect(sh).toContain('PORT')
    expect(sh).toContain('HTTP_ADDR')
    expect(sh).toContain('HTTP_ADDR="${HTTP_ADDR:-0.0.0.0:${PORT:-30808}}"')
    expect(sh).toContain('PUBLIC_URL')
    expect(sh).not.toContain('127.0.0.1:8082')
    expect(sh).toContain('go run ./apps/pixoma/cmd/pixoma')
    expect(sh).toContain('VITE_ADMIN_API_BASE=')
    expect(sh).toContain('pnpm --dir web/admin dev')
    expect(sh).toMatch(/trap .* INT TERM/)
  })
})

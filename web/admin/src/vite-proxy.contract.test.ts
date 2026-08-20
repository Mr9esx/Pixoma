import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const adminRoot = join(dirname(fileURLToPath(import.meta.url)), '..')

describe('vite admin API proxy', () => {
  it('proxies /api to pixoma 8082', () => {
    const src = readFileSync(join(adminRoot, 'vite.config.ts'), 'utf8')
    expect(src).toMatch(/proxy\s*:/)
    expect(src).toMatch(/host\s*:\s*['"]127\.0\.0\.1['"]/)
    expect(src).toMatch(/['"]\/api['"]\s*:/)
    expect(src).toMatch(/target\s*:\s*['"]http:\/\/127\.0\.0\.1:8082['"]/)
    expect(src).toMatch(/changeOrigin\s*:\s*true/)
    expect(src).toMatch(/command\s*===\s*['"]serve['"]/)
    expect(src).toMatch(
      /import\.meta\.env\.VITE_ADMIN_API_BASE['"]\s*:\s*JSON\.stringify\(['"]{2}\)/,
    )
  })
})

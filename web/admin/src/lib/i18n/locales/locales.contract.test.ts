import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function readLocale(name: 'en.json' | 'zh.json') {
  return JSON.parse(readFileSync(join(here, name), 'utf8')) as {
    common?: Record<string, unknown>
  }
}

describe('locale resources', () => {
  it('provides the shared action column label', () => {
    expect(readLocale('zh.json').common?.actions).toBe('操作')
    expect(readLocale('en.json').common?.actions).toBe('Actions')
  })
})

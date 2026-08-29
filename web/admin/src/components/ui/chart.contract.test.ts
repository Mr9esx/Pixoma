import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const CHART = join(here, 'chart.tsx')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('chart container resize debounce', () => {
  it('throttles ResponsiveContainer resize to stay smooth during sidebar animation', () => {
    const source = read(CHART)
    expect(source).toContain('CHART_RESIZE_DEBOUNCE')
    expect(source).toMatch(/debounce = CHART_RESIZE_DEBOUNCE/)
    expect(source).toMatch(
      /<RechartsPrimitive\.ResponsiveContainer[^>]*debounce=\{debounce\}[^>]*>/
    )
  })
})

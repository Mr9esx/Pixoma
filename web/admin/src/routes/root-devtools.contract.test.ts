import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const root = readFileSync(join(here, '__root.tsx'), 'utf8')

describe('root devtools', () => {
  it('hides the TanStack Query floating open button', () => {
    expect(root).not.toContain('ReactQueryDevtools')
    expect(root).not.toContain('buttonPosition')
  })

  it('keeps router devtools at the bottom-right corner', () => {
    expect(root).toContain('TanStackRouterDevtools')
    expect(root).toContain("position='bottom-right'")
  })
})

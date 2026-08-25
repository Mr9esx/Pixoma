import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('session list panel', () => {
  it('renders the empty state inside the table body', () => {
    const source = read('list-panel.tsx')
    expect(source).toMatch(/<tbody/)
    expect(source).toMatch(/colSpan=\{7\}/)
    expect(source).toMatch(
      /EmptyState[\s\S]*?className='py-8'[\s\S]*?message=\{t\('sessions\.empty'\)\}/
    )
  })
})

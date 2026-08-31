import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))

function read(rel: string) {
  return readFileSync(join(here, rel), 'utf8')
}

describe('session list panel', () => {
  it('renders a tablecn data table with empty state by i18n key', () => {
    const source = read('list-panel.tsx')
    expect(source).toMatch(/data-table\/data-table'/)
    expect(source).toMatch(
      /<EmptyState className='py-8' message=\{t\('sessions\.empty'\)\} \/>/
    )
  })

  it('shows platform context', () => {
    const source = read('list-panel.tsx')
    expect(source).toMatch(/fieldPlatform/)
    expect(source).toMatch(/channel_name \|\|/)
    expect(read('detail-panel.tsx')).toMatch(/fieldPlatform/)
  })
})

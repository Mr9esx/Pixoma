import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const LIST_PANEL = join(here, 'list-panel.tsx')
const CASES_ROUTE = join(here, '../../routes/_app/cases/route.tsx')
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('case list filters (single search, no status segment)', () => {
  it('list panel has one search field and no menu_key input', () => {
    const source = read(LIST_PANEL)
    expect(source).toContain("id='cases-filter-q'")
    expect(source).not.toContain('cases-filter-menu-key')
    expect(source).not.toContain('filterMenuKey')
    expect(source).not.toMatch(/menuKey/)
  })

  it('no status filter or status badge remains', () => {
    const source = read(LIST_PANEL)
    expect(source).not.toContain('cases-filter-enabled')
    expect(source).not.toContain('FilterSegment')
    expect(source).not.toContain('item.enabled')
    expect(source).not.toContain("t('cases.enabled')")
  })

  it('list item title is the workflow name and secondary line is the description', () => {
    const source = read(LIST_PANEL)
    expect(source).toMatch(/font-medium'[^>]*>\s*\{item\.name\}/)
    expect(source).toContain('{item.description}')
    expect(source).not.toMatch(/font-medium'[^>]*>\{item\.id\}/)
  })

  it('cases route does not send menu_key query param from filters', () => {
    const source = read(CASES_ROUTE)
    expect(source).not.toMatch(/menuKey/)
    expect(source).not.toMatch(/menu_key:\s*filters/)
  })

  it('cases route auto-selects the first workflow and navigates the url', () => {
    const source = read(CASES_ROUTE)
    expect(source).toContain('items[0]?.id')
    expect(source).toContain('replace: true')
    expect(source).toContain('backToList')
  })

  it('search placeholder does not mention menu_key', () => {
    const zh = JSON.parse(read(ZH)) as {
      cases: { filterQPlaceholder: string }
    }
    const en = JSON.parse(read(EN)) as {
      cases: { filterQPlaceholder: string }
    }
    expect(zh.cases.filterQPlaceholder.toLowerCase()).not.toContain('menu_key')
    expect(en.cases.filterQPlaceholder.toLowerCase()).not.toContain('menu_key')
  })
})

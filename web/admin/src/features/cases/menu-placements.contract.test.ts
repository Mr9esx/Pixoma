import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const API = join(here, '../../lib/api/channel-menu.ts')
const SECTION = join(here, 'sections/menu-placements.tsx')
const DETAIL = join(here, 'detail-panel.tsx')
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('case menu placements section', () => {
  it('api exposes getCaseMenuPlacements', () => {
    const source = read(API)
    expect(source).toContain('export type MenuPlacement')
    expect(source).toContain('export function getCaseMenuPlacements')
    expect(source).toContain('/menu-placements')
  })

  it('section joins path labels and shows empty state', () => {
    const source = read(SECTION)
    expect(source).toContain("join(' / ')")
    expect(source).toContain('cases.menuPlacementsEmpty')
    expect(source).toContain('getCaseMenuPlacements')
    expect(source).toContain('queryKeys.cases.menuPlacements')
  })

  it('detail panel renders menu placements section', () => {
    const source = read(DETAIL)
    expect(source).toContain('MenuPlacementsSection')
    expect(source).toContain('caseId={record.id}')
  })

  it('i18n has menu placement keys', () => {
    const zh = JSON.parse(read(ZH)) as {
      cases: { menuPlacementsTitle: string; menuPlacementsEmpty: string }
    }
    const en = JSON.parse(read(EN)) as {
      cases: { menuPlacementsTitle: string; menuPlacementsEmpty: string }
    }
    expect(zh.cases.menuPlacementsTitle).toBeTruthy()
    expect(zh.cases.menuPlacementsEmpty).toBeTruthy()
    expect(en.cases.menuPlacementsTitle).toBeTruthy()
    expect(en.cases.menuPlacementsEmpty).toBeTruthy()
  })
})

import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const srcRoot = join(here, '../..')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

const FILTER_SEGMENT = join(here, 'filter-segment.tsx')
const PRINCIPLES = join(
  srcRoot,
  '../../../docs/frontend/admin-list-filters.md',
)

describe('admin list filter principles', () => {
  it('documents compact list filter principles', () => {
    const doc = read(PRINCIPLES)
    expect(doc).toContain('横向可滑')
    expect(doc).toContain('FilterSegment')
    expect(doc).toContain('并进一个搜索框')
  })

  it('FilterSegment uses chevron buttons inside the segment (no scrollbar)', () => {
    const source = read(FILTER_SEGMENT)
    expect(source).toContain('overflow-x-auto')
    expect(source).toContain('[scrollbar-width:none]')
    expect(source).toContain('ChevronLeft')
    expect(source).toContain('ChevronRight')
    expect(source).toContain('rounded-full')
    expect(source).toContain('transition-opacity duration-300')
    expect(source).toContain('left-1.5')
    expect(source).toContain('right-1.5')
    expect(source).toContain('FILL_BELOW')
    expect(source).toContain('flex-1 text-center')
    expect(source).toContain('scrollBy')
    expect(source).toContain('[value, fill]')
    expect(source).not.toContain('padLeft')
    expect(source).not.toContain('transition-[padding]')
    expect(source).not.toContain('canScrollLeft, canScrollRight')
  })

  it('cases / tasks / sessions use FilterSegment; no Select in list panels', () => {
    for (const rel of [
      'features/cases/list-panel.tsx',
      'features/tasks/list-panel.tsx',
      'features/sessions/list-panel.tsx',
    ]) {
      const source = read(join(srcRoot, rel))
      expect(source).toContain("from '@/components/filters/filter-segment'")
      expect(source).not.toContain('<Select')
      expect(source).not.toContain('<Label')
    }
  })

  it('users list has single search and no tg_user_id filter field', () => {
    const panel = read(join(srcRoot, 'features/users/list-panel.tsx'))
    const route = read(join(srcRoot, 'routes/_app/users/route.tsx'))
    expect(panel).toContain("id='users-filter-q'")
    expect(panel).not.toContain('users-filter-tg')
    expect(panel).not.toContain('filterTgUserId')
    expect(route).not.toMatch(/tg_user_id:/)
  })

  it('sessions list has no dedicated user_id filter field', () => {
    const panel = read(join(srcRoot, 'features/sessions/list-panel.tsx'))
    const route = read(join(srcRoot, 'routes/_app/sessions/route.tsx'))
    expect(panel).not.toContain('sessions-filter-user')
    expect(route).not.toMatch(/user_id:\s*filters/)
  })
})

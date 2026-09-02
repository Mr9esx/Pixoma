import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const src = () => readFileSync(join(here, 'menu-writable-map.tsx'), 'utf8')

describe('MenuWritableMap', () => {
  it('edits key labels in the path pane and actions only there', () => {
    const s = src()
    expect(s).toContain("data-testid='map-keyboard'")
    expect(s).toContain("data-testid='add-key'")
    expect(s).toContain("data-testid='map-key-label'")
    expect(s).toContain("t('menu.keyName')")
    expect(s).toContain('ActionForm')
    expect(s).toContain("t('menu.actionRow')")
    expect(s).toContain('trailFromItem')
    expect(s).toContain('pushTrailButton')
    expect(s).toContain("kind === 'orphan'")
    expect(s).toContain("data-testid='menu-columns'")
    expect(s).toContain("t('menu.deleteKey')")
    expect(s).toContain("t('menu.deleteButton')")
    expect(s).toContain("t('menu.deleteCard')")
  })

  it('edits open_card as a message and pushes trail on button click', () => {
    const s = src()
    expect(s).toContain("data-testid='edit-card'")
    expect(s).toContain("data-testid='add-button'")
    expect(s).toContain("data-testid='edit-card-button'")
    expect(s).toContain('pushTrailButton')
    expect(s).toContain('buildNewCardDraft')
    expect(s).toContain("data-testid='new-card-name'")
    expect(s).toContain("data-testid='new-card-confirm'")
    expect(s).not.toContain('NewCardDialog')
    expect(s).not.toContain('ButtonEditor')
  })

  it('column picker is compact FilterSegment; keys show label only', () => {
    const s = src()
    expect(s).toContain('FilterSegment')
    expect(s).toContain("data-testid='menu-columns'")
    expect(s).not.toContain('grid-cols-8')
    expect(s).not.toMatch(/data-testid='map-key'[\s\S]{0,800}outcomeText/)
  })

  it('lists orphans with map-orphans and can delete a card', () => {
    const s = src()
    expect(s).toContain("data-testid='map-orphans'")
    expect(s).toContain('orphanCards')
    expect(s).toContain('trailFromOrphan')
    expect(s).toContain("data-testid='delete-card'")
  })
})

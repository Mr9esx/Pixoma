import { describe, expect, it } from 'vitest'
import type { MenuTree } from '@/lib/api/channel-menu'
import { validateMenuTree } from './validate-tree'

const ok: MenuTree = {
  id: 'ch1',
  columns: 2,
  items: [
    {
      id: 'a',
      label: '帮助',
      action: { type: 'send_text', text: 'hi' },
    },
  ],
}

describe('validateMenuTree', () => {
  it('accepts a minimal legal tree', () => {
    expect(validateMenuTree(ok)).toEqual([])
  })

  it('rejects empty root and missing nested card', () => {
    expect(
      validateMenuTree({ id: 'ch1', columns: 2, items: [] }).some(
        (i) => i.key === 'menu.errRootCount'
      )
    ).toBe(true)
    expect(
      validateMenuTree({
        id: 'ch1',
        columns: 2,
        items: [{ id: 'a', label: '开', action: { type: 'open_card' } }],
      }).some((i) => i.key === 'menu.errCard')
    ).toBe(true)
  })
})

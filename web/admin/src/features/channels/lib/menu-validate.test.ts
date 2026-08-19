import { describe, expect, it } from 'vitest'
import type { MenuNode } from '@/lib/api/channel-menu'
import { countUnsaved, validateMenu } from './menu-validate'

function node(partial: Partial<MenuNode>): MenuNode {
  return { id: 'x', label: 'x', order: 0, enabled: true, ...partial }
}

describe('menu validate', () => {
  it('rejects more than 6 root buttons', () => {
    const items = Array.from({ length: 7 }, (_, i) =>
      node({
        id: `r${i}`,
        label: `R${i}`,
        order: i,
        capability_id: 'reply_text',
        params: { text: 'x' },
      })
    )
    expect(validateMenu(items).ok).toBe(false)
  })

  it('rejects leaf without capability and open_case without cases', () => {
    const items: MenuNode[] = [
      node({ id: 'a', label: 'A', order: 0 }),
      node({
        id: 'b',
        label: 'B',
        order: 1,
        capability_id: 'open_case',
        params: { case_ids: [] },
      }),
    ]
    const errors = validateMenu(items).errors
    expect(errors.some((e) => e.id === 'a')).toBe(true)
    expect(errors.some((e) => e.id === 'b')).toBe(true)
  })

  it('rejects nested groups', () => {
    const items: MenuNode[] = [
      node({
        id: 'g',
        label: 'G',
        order: 0,
        children: [
          node({
            id: 'g2',
            label: 'G2',
            order: 0,
            children: [
              node({
                id: 'x',
                label: 'X',
                order: 0,
                capability_id: 'reply_text',
                params: { text: 'x' },
              }),
            ],
          }),
        ],
      }),
    ]
    expect(validateMenu(items).ok).toBe(false)
  })

  it('counts unsaved changes against baseline', () => {
    const baseline: MenuNode[] = [
      node({
        id: 'a',
        label: 'A',
        order: 0,
        capability_id: 'reply_text',
        params: { text: 'x' },
      }),
    ]
    const changed: MenuNode[] = [
      node({
        id: 'a',
        label: 'A2',
        order: 0,
        capability_id: 'reply_text',
        params: { text: 'x' },
      }),
    ]
    expect(countUnsaved(changed, baseline)).toBe(1)
    expect(countUnsaved(baseline, baseline)).toBe(0)
  })
})

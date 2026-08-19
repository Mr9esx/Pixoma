import { describe, expect, it } from 'vitest'
import type { MenuNode } from '@/lib/api/channel-menu'
import { buildTgSimulation, rootColumnsOf } from './menu-simulation'

function node(partial: Partial<MenuNode>): MenuNode {
  return { id: 'x', label: 'x', order: 0, enabled: true, ...partial }
}

describe('menu simulation', () => {
  it('builds keyboard rows from enabled root items with columns', () => {
    const items: MenuNode[] = [
      node({
        id: 'a',
        label: 'A',
        order: 0,
        render_override: { columns: 2 },
      }),
      node({ id: 'b', label: 'B', order: 1 }),
      node({ id: 'c', label: 'C', order: 2 }),
      node({ id: 'off', label: 'OFF', order: 3, enabled: false }),
    ]
    expect(rootColumnsOf(items)).toBe(2)
    expect(buildTgSimulation(items).keyboard).toEqual([['A', 'B'], ['C']])
  })

  it('renders one-level groups with intro and child buttons', () => {
    const items: MenuNode[] = [
      node({
        id: 'tools',
        label: '工具',
        order: 0,
        intro_text: '小工具们',
        children: [
          node({
            id: 'edit',
            label: 'AI 修图',
            order: 0,
            capability_id: 'open_case',
            params: { case_ids: ['c1'] },
          }),
          node({
            id: 'matting',
            label: '抠图',
            order: 1,
            capability_id: 'open_case',
            params: { case_ids: ['c2'] },
          }),
        ],
      }),
    ]
    const sim = buildTgSimulation(items)
    expect(sim.groups['tools']).toEqual({
      id: 'tools',
      title: '工具',
      intro: '小工具们',
      buttons: [
        { label: 'AI 修图', kind: 'capability', capabilityId: 'open_case' },
        { label: '抠图', kind: 'capability', capabilityId: 'open_case' },
      ],
    })
  })

  it('filters disabled children from group buttons', () => {
    const items: MenuNode[] = [
      node({
        id: 'g',
        label: 'G',
        order: 0,
        children: [
          node({
            id: 'on',
            label: 'ON',
            order: 0,
            capability_id: 'reply_text',
            params: { text: 'x' },
          }),
          node({
            id: 'off',
            label: 'OFF',
            order: 1,
            enabled: false,
            capability_id: 'reply_text',
            params: { text: 'x' },
          }),
        ],
      }),
    ]
    expect(
      buildTgSimulation(items).groups['g'].buttons.map((b) => b.label)
    ).toEqual(['ON'])
  })
})

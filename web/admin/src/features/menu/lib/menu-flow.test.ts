import { describe, expect, it } from 'vitest'
import {
  buildTgFlow,
  validateMenuConfig,
  type Action,
  type Card,
  type Menu,
} from './menu-flow'

const placeholder: Action = { type: 'placeholder' }

describe('menu flow', () => {
  it('builds keyboard rows without a 6-button cap', () => {
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: Array.from({ length: 7 }, (_, i) => ({
        id: `i${i}`,
        label: `B${i}`,
        action: placeholder,
      })),
    }
    const { keyboard } = buildTgFlow(menu, [])
    expect(keyboard.flat().length).toBe(7)
    expect(keyboard[0]).toEqual(['B0', 'B1'])
  })

  it('collects cards reachable from menu and buttons', () => {
    const cardA: Card = {
      id: 'a',
      name: 'A',
      media: [],
      text: 'A',
      buttons: [
        { id: 'b1', label: '去B', action: { type: 'open_card', card_id: 'b' } },
      ],
    }
    const cardB: Card = {
      id: 'b',
      name: 'B',
      media: [],
      text: 'B',
      buttons: [],
    }
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        {
          id: 'mi',
          label: '入口',
          action: { type: 'open_card', card_id: 'a' },
        },
      ],
    }
    const flow = buildTgFlow(menu, [cardA, cardB])
    expect(flow.cardById.get('a')).toEqual(cardA)
    expect(flow.cardById.get('b')).toEqual(cardB)
  })

  it('validates only on save: missing label, action target, workflow ids', () => {
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        { id: 'x', label: '', action: placeholder },
        { id: 'y', label: 'Y', action: { type: 'open_card' } },
        {
          id: 'z',
          label: 'Z',
          action: { type: 'open_workflow', workflow_ids: [] },
        },
      ],
    }
    const errors = validateMenuConfig(menu, []).errors
    expect(errors.length).toBeGreaterThanOrEqual(3)
  })
})

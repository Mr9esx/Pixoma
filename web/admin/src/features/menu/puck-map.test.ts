import { describe, expect, it } from 'vitest'
import type { MenuTree } from '@/lib/api/channel-menu'
import { fromPuckData, toPuckData } from './puck-map'

describe('puckData ↔ MenuTree', () => {
  it('round-trips nested open_card', () => {
    const tree: MenuTree = {
      id: 'ch1',
      columns: 2,
      items: [
        {
          id: 'kbd-1',
          label: '风格',
          action: {
            type: 'open_card',
            card: {
              text: '选风格',
              buttons: [
                {
                  id: 'b1',
                  label: '写实',
                  action: { type: 'open_workflow', workflow_id: '10' },
                },
                {
                  id: 'b2',
                  label: '再开',
                  action: {
                    type: 'open_card',
                    card: {
                      text: '内层',
                      buttons: [
                        {
                          id: 'b3',
                          label: '内层键',
                          action: { type: 'send_text', text: 'hi' },
                        },
                      ],
                    },
                  },
                },
              ],
            },
          },
        },
      ],
    }
    const back = fromPuckData(toPuckData(tree), 'ch1')
    expect(back.columns).toBe(2)
    expect(back.items[0].label).toBe('风格')
    expect(back.items[0].action.card?.text).toBe('选风格')
    expect(back.items[0].action.card?.buttons?.[0].action.workflow_id).toBe('10')
    expect(back.items[0].action.card?.buttons?.[1].action.card?.text).toBe('内层')
    expect(back.items[0].action.card?.buttons?.[1].action.card?.buttons?.[0].label).toBe(
      '内层键'
    )
  })
})
import { describe, expect, it } from 'vitest'
import type { MenuTree } from '@/lib/api/channel-menu'
import { walkMenuWorkflows } from './walk-menu-workflows'

describe('walkMenuWorkflows', () => {
  it('collects open_workflow ids from root buttons', () => {
    const menu: MenuTree = {
      id: 'm',
      columns: 2,
      items: [
        {
          id: 'a',
          label: 'A',
          action: { type: 'open_workflow', workflow_id: '10' },
        },
        { id: 'b', label: 'B', action: { type: 'send_text', text: 'hi' } },
      ],
    }
    expect(walkMenuWorkflows(menu)).toEqual(['10'])
  })

  it('walks nested card buttons and dedupes', () => {
    const menu: MenuTree = {
      id: 'm',
      columns: 1,
      items: [
        {
          id: 'card',
          label: 'Card',
          action: {
            type: 'open_card',
            card: {
              text: 'x',
              buttons: [
                {
                  id: 'n',
                  label: 'N',
                  action: { type: 'open_workflow', workflow_id: '10' },
                },
                {
                  id: 'n2',
                  label: 'N2',
                  action: { type: 'open_workflow', workflow_id: '11' },
                },
              ],
            },
          },
        },
        {
          id: 'dup',
          label: 'Dup',
          action: { type: 'open_workflow', workflow_id: '10' },
        },
      ],
    }
    expect(walkMenuWorkflows(menu)).toEqual(['10', '11'])
  })
})

import { describe, expect, it } from 'vitest'
import type { WorkflowRef } from '../node-view'
import {
  actionIsBroken,
  actionOutcomeLabel,
  buildTgFlow,
  clampMenuColumns,
  createBlankCardButton,
  createBlankMenuItem,
  isMenuDraftDirty,
  mediaKindLabelKey,
  orphanCards,
  pushTrailButton,
  reachableCardIds,
  sliceTrail,
  trailFromItem,
  trailFromOrphan,
  validateMenuConfig,
  workflowEntryCount,
  type Action,
  type Card,
  type CardButton,
  type Menu,
  type MenuItem,
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

describe('validateAction (v2 schema)', () => {
  it('open_workflow with workflow_id passes (no mode required)', () => {
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        {
          id: 'i1',
          label: 'B1',
          action: { type: 'open_workflow', workflow_id: '10' } as Action,
        },
      ],
    }
    const result = validateMenuConfig(menu, [])
    expect(result.ok).toBe(true)
    expect(result.errors).toEqual([])
  })

  it('open_workflow without workflow_id reports workflow error', () => {
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        { id: 'i1', label: 'B1', action: { type: 'open_workflow' } as Action },
      ],
    }
    const result = validateMenuConfig(menu, [])
    expect(result.ok).toBe(false)
    expect(result.errors.some((e) => e.message === 'workflow')).toBe(true)
  })

  it('validate ignores unknown mode field (no schema-level rejection)', () => {
    // 即使 Action 携带 mode 字段（旧数据），validate 也不应把它当 error。
    const legacy = {
      type: 'open_workflow',
      workflow_id: '10',
      mode: 'list',
    } as Action
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [{ id: 'i1', label: 'B1', action: legacy }],
    }
    const result = validateMenuConfig(menu, [])
    expect(result.ok).toBe(true)
  })
})

const workflows: WorkflowRef[] = [
  { id: 1, name: '写实人像' },
  { id: 2, name: '动漫风' },
]

const style: Card = {
  id: 'style',
  name: '风格选择',
  media: [],
  text: '选',
  buttons: [
    {
      id: 'b1',
      label: '写实',
      action: { type: 'open_workflow', workflow_id: '1' },
    },
    {
      id: 'b2',
      label: '说明',
      action: { type: 'open_card', card_id: 'guide' },
    },
  ],
}
const guide: Card = {
  id: 'guide',
  name: '风格说明',
  media: [],
  text: '说明',
  buttons: [],
}
const welcome: Card = {
  id: 'welcome',
  name: '欢迎卡',
  media: [],
  text: '未挂',
  buttons: [
    {
      id: 'w1',
      label: '去生成',
      action: { type: 'open_card', card_id: 'style' },
    },
  ],
}
const cycleA: Card = {
  id: 'ca',
  name: 'A',
  media: [],
  text: 'A',
  buttons: [
    { id: 'x', label: '去B', action: { type: 'open_card', card_id: 'cb' } },
  ],
}
const cycleB: Card = {
  id: 'cb',
  name: 'B',
  media: [],
  text: 'B',
  buttons: [
    { id: 'y', label: '去A', action: { type: 'open_card', card_id: 'ca' } },
  ],
}

describe('capability map helpers', () => {
  it('actionOutcomeLabel 六种动作与断引用', () => {
    expect(
      actionOutcomeLabel(
        { type: 'open_card', card_id: 'style' },
        [style],
        workflows
      )
    ).toEqual({
      key: 'mapOpenCard',
      name: '风格选择',
    })
    expect(
      actionOutcomeLabel(
        { type: 'open_card', card_id: 'gone' },
        [style],
        workflows
      )
    ).toEqual({
      key: 'mapCardMissing',
    })
    expect(
      actionOutcomeLabel(
        { type: 'open_workflow', workflow_id: '1' },
        [],
        workflows
      )
    ).toEqual({
      key: 'mapStartWorkflow',
      name: '写实人像',
    })
    expect(
      actionOutcomeLabel(
        { type: 'open_workflow', workflow_id: 'gone' },
        [],
        workflows
      )
    ).toEqual({
      key: 'mapWorkflowMissing',
    })
    expect(
      actionOutcomeLabel({ type: 'send_text', text: 'hi' }, [], [])
    ).toEqual({ key: 'mapSendText' })
    expect(
      actionOutcomeLabel({ type: 'send_media', media: [] }, [], [])
    ).toEqual({ key: 'mapSendMedia' })
    expect(
      actionOutcomeLabel({ type: 'open_url', url: 'https://x' }, [], [])
    ).toEqual({ key: 'mapOpenUrl' })
    expect(
      actionOutcomeLabel({ type: 'copy_text', text: 'CODE' }, [], [])
    ).toEqual({ key: 'mapCopyText' })
  })

  it('actionIsBroken 只在卡片/工作流缺失时为 true', () => {
    expect(
      actionIsBroken({ type: 'open_card', card_id: 'gone' }, [style], workflows)
    ).toBe(true)
    expect(
      actionIsBroken(
        { type: 'open_workflow', workflow_id: 'gone' },
        [],
        workflows
      )
    ).toBe(true)
    expect(
      actionIsBroken(
        { type: 'open_card', card_id: 'style' },
        [style],
        workflows
      )
    ).toBe(false)
    expect(actionIsBroken({ type: 'send_text', text: 'hi' }, [], [])).toBe(
      false
    )
  })

  it('reachableCardIds 从主键盘沿 open_card 走，循环不死', () => {
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        {
          id: 'i1',
          label: '图',
          action: { type: 'open_card', card_id: 'style' },
        },
      ],
    }
    const ids = reachableCardIds(menu, [style, guide, welcome])
    expect([...ids].sort()).toEqual(['guide', 'style'])
    const cyc: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        { id: 'i', label: 'A', action: { type: 'open_card', card_id: 'ca' } },
      ],
    }
    expect(reachableCardIds(cyc, [cycleA, cycleB]).size).toBe(2)
  })

  it('orphanCards 是键盘走不到的卡片', () => {
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        {
          id: 'i1',
          label: '图',
          action: { type: 'open_card', card_id: 'style' },
        },
      ],
    }
    expect(orphanCards(menu, [style, guide, welcome]).map((c) => c.id)).toEqual(
      ['welcome']
    )
  })

  it('workflowEntryCount 去重且计入断引用与未挂上卡片上的入口', () => {
    const menu: Menu = {
      id: 'm',
      name: '主',
      columns: 2,
      items: [
        {
          id: 'i1',
          label: '直出',
          action: { type: 'open_workflow', workflow_id: '1' },
        },
        {
          id: 'i2',
          label: '会员',
          action: { type: 'open_workflow', workflow_id: 'gone' },
        },
      ],
    }
    expect(workflowEntryCount(menu, [style, welcome])).toBe(2)
  })

  it('mediaKindLabelKey', () => {
    expect(mediaKindLabelKey('image')).toBe('mapThumbImage')
    expect(mediaKindLabelKey('video')).toBe('mapThumbVideo')
    expect(mediaKindLabelKey('animation')).toBe('mapThumbAnimation')
  })

  it('trailFromItem / pushTrailButton / sliceTrail', () => {
    const item: MenuItem = {
      id: 'i1',
      label: '图片生成',
      action: { type: 'open_card', card_id: 'style' },
    }
    const t0 = trailFromItem(item)
    expect(t0).toEqual([
      { kind: 'item', id: 'i1', label: '图片生成', action: item.action },
    ])
    const btn: CardButton = {
      id: 'b1',
      label: '写实',
      action: { type: 'open_workflow', workflow_id: '1' },
    }
    const t1 = pushTrailButton(t0, btn)
    expect(t1).toHaveLength(2)
    expect(t1[1]).toEqual({
      kind: 'btn',
      id: 'b1',
      label: '写实',
      action: btn.action,
    })
    expect(sliceTrail(t1, 0)).toEqual(t0)
  })

  it('clampMenuColumns keeps 1–8, defaults 2', () => {
    expect(clampMenuColumns(1)).toBe(1)
    expect(clampMenuColumns(8)).toBe(8)
    expect(clampMenuColumns(0)).toBe(2)
    expect(clampMenuColumns(9)).toBe(2)
    expect(clampMenuColumns(3.7)).toBe(3)
  })

  it('isMenuDraftDirty is false when equal, true when columns or a label changes', () => {
    const menu: Menu = {
      id: 'ch',
      name: '',
      columns: 2,
      items: [
        { id: 'i1', label: 'A', action: { type: 'send_text', text: 'x' } },
      ],
    }
    const cards: Card[] = []
    expect(isMenuDraftDirty({ menu, cards }, { menu, cards })).toBe(false)
    expect(
      isMenuDraftDirty(
        { menu, cards },
        { menu: { ...menu, columns: 3 }, cards }
      )
    ).toBe(true)
    expect(
      isMenuDraftDirty(
        { menu, cards },
        {
          menu: {
            ...menu,
            items: [{ ...menu.items[0], label: 'B' }],
          },
          cards,
        }
      )
    ).toBe(true)
  })

  it('createBlankMenuItem defaults to empty send_text; createBlankCardButton same', () => {
    const item = createBlankMenuItem('mi-1')
    expect(item).toEqual({
      id: 'mi-1',
      label: '',
      action: { type: 'send_text', text: '' },
    })
    const btn = createBlankCardButton('b-1')
    expect(btn).toEqual({
      id: 'b-1',
      label: '',
      action: { type: 'send_text', text: '' },
    })
  })

  it('trailFromOrphan 起点是 orphan + open_card 自己', () => {
    const t = trailFromOrphan(welcome)
    expect(t).toEqual([
      {
        kind: 'orphan',
        id: 'welcome',
        label: '欢迎卡',
        action: { type: 'open_card', card_id: 'welcome' },
      },
    ])
  })
})

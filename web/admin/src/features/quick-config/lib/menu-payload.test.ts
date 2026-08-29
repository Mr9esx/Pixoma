import { describe, expect, it } from 'vitest'
import type { Menu } from '@/lib/api/channel-menu'
import { addWorkflowMenuEntry } from './menu-payload'

const baseMenu: Menu = {
  id: 'default',
  name: '主菜单',
  columns: 2,
  items: [{ id: 'mi-help', label: '帮助', action: { type: 'send_text', text: '帮助菜单' } }],
}

describe('addWorkflowMenuEntry (v2 schema)', () => {
  it('追加 open_workflow 菜单项用 workflow_id 单数且无 mode', () => {
    const next = addWorkflowMenuEntry(baseMenu, {
      label: '🖼 图片生成',
      workflowId: 12,
    })
    expect(baseMenu.items).toHaveLength(1)
    expect(next.items).toHaveLength(2)
    const item = next.items[1]
    expect(item.label).toBe('🖼 图片生成')
    expect(item.action).toEqual({
      type: 'open_workflow',
      workflow_id: '12',
    })
    expect(item.id).toBeTruthy()
  })

  it('空标签抛出错误', () => {
    expect(() =>
      addWorkflowMenuEntry(baseMenu, {
        label: '  ',
        workflowId: 12,
      })
    ).toThrow()
  })
})

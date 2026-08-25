import { describe, expect, it } from 'vitest'
import type { Menu } from '@/lib/api/channel-menu'
import { addWorkflowMenuEntry } from './menu-payload'

const baseMenu: Menu = {
  id: 'default',
  name: '主菜单',
  columns: 2,
  items: [{ id: 'mi-help', label: '帮助', action: { type: 'placeholder' } }],
}

describe('addWorkflowMenuEntry', () => {
  it('追加 open_workflow 菜单项且不修改原菜单', () => {
    const next = addWorkflowMenuEntry(baseMenu, {
      label: '🖼 图片生成',
      mode: 'direct',
      workflowId: 12,
    })
    expect(baseMenu.items).toHaveLength(1)
    expect(next.items).toHaveLength(2)
    const item = next.items[1]
    expect(item.label).toBe('🖼 图片生成')
    expect(item.action).toEqual({
      type: 'open_workflow',
      workflow_ids: ['12'],
      mode: 'direct',
    })
    expect(item.id).toBeTruthy()
  })

  it('list 模式保留 mode 字段', () => {
    const next = addWorkflowMenuEntry(baseMenu, {
      label: '全部工作流',
      mode: 'list',
      workflowId: 12,
    })
    expect(next.items[1].action).toEqual({
      type: 'open_workflow',
      workflow_ids: ['12'],
      mode: 'list',
    })
  })

  it('空标签抛出错误', () => {
    expect(() =>
      addWorkflowMenuEntry(baseMenu, {
        label: '  ',
        mode: 'direct',
        workflowId: 12,
      })
    ).toThrow()
  })
})

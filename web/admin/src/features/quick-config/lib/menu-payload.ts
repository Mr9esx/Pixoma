import type { Menu, MenuItem } from '@/lib/api/channel-menu'

export type WorkflowMenuMode = 'direct' | 'list'

/**
 * 在渠道菜单末尾追加一个「打开指定工作流」的主菜单按钮。
 * 保持原菜单不变（不可变更新），动作类型锁定 open_workflow。
 */
export function addWorkflowMenuEntry(
  menu: Menu,
  entry: { label: string; mode: WorkflowMenuMode; workflowId: number }
): Menu {
  const label = entry.label.trim()
  if (!label) {
    throw new Error('menu item label is required')
  }
  const item: MenuItem = {
    id: `qc-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    label,
    action: {
      type: 'open_workflow',
      workflow_ids: [entry.workflowId],
      mode: entry.mode,
    },
  }
  return { ...menu, items: [...menu.items, item] }
}

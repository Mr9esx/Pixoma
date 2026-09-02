import type { Menu, MenuItem } from '@/lib/api/channel-menu'

const MAX_ROOT_BUTTONS = 6

/**
 * 在消息平台菜单末尾追加一个「打开指定工作流」的主菜单按钮。
 * 保持原菜单不变（不可变更新），动作类型锁定 open_workflow，
 * 关联单个工作流 id（v2 schema：`workflow_id` 单数字段，无 mode）。
 */
export function addWorkflowMenuEntry(
  menu: Menu,
  entry: { label: string; workflowId: number }
): Menu {
  const label = entry.label.trim()
  if (!label) {
    throw new Error('menu item label is required')
  }
  if (menu.items.length >= MAX_ROOT_BUTTONS) {
    throw new Error('root keyboard is full')
  }
  const item: MenuItem = {
    id: `qc-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    label,
    action: {
      type: 'open_workflow',
      workflow_id: String(entry.workflowId),
    },
  }
  return { ...menu, items: [...menu.items, item] }
}

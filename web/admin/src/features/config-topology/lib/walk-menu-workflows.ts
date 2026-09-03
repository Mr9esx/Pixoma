import type { MenuAction, MenuButton, MenuTree } from '@/lib/api/channel-menu'

function collect(action: MenuAction | undefined, into: string[]) {
  if (!action) return
  if (action.type === 'open_workflow' && action.workflow_id) {
    into.push(action.workflow_id)
  }
  for (const button of action.card?.buttons ?? []) {
    collectFromButton(button, into)
  }
}

function collectFromButton(button: MenuButton, into: string[]) {
  collect(button.action, into)
}

export function walkMenuWorkflows(menu: MenuTree): string[] {
  const ids: string[] = []
  for (const item of menu.items) {
    collectFromButton(item, ids)
  }
  return [...new Set(ids)]
}

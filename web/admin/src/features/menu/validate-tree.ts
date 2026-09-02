import type { MenuAction, MenuButton, MenuCard, MenuTree } from '@/lib/api/channel-menu'

const MAX_ROOT = 6
const MAX_DEPTH = 8

export type TreeIssue = { key: string }

function hasHTTPPrefix(url: string): boolean {
  return url.startsWith('http://') || url.startsWith('https://')
}

function validateAction(
  action: MenuAction,
  depth: number,
  seen: Set<string>,
  issues: TreeIssue[],
  workflowIds?: Set<string>
): void {
  switch (action.type) {
    case 'open_card':
      if (!action.card) {
        issues.push({ key: 'menu.errCard' })
        return
      }
      if (depth + 1 > MAX_DEPTH) {
        issues.push({ key: 'menu.errDepth' })
        return
      }
      validateCard(action.card, depth + 1, seen, issues, workflowIds)
      return
    case 'open_workflow':
      if (!action.workflow_id) {
        issues.push({ key: 'menu.errWorkflow' })
        return
      }
      if (workflowIds && !workflowIds.has(action.workflow_id)) {
        issues.push({ key: 'menu.errWorkflow' })
      }
      return
    case 'list_tasks':
      return
    case 'open_url':
      if (!hasHTTPPrefix(action.url ?? '')) {
        issues.push({ key: 'menu.errUrl' })
      }
      return
    case 'send_media':
      if (!action.media?.length) {
        issues.push({ key: 'menu.errMedia' })
        return
      }
      for (const m of action.media) {
        if (!hasHTTPPrefix(m.url)) {
          issues.push({ key: 'menu.errMedia' })
          return
        }
      }
      return
    case 'send_text':
    case 'copy_text':
      return
    default:
      issues.push({ key: 'menu.saveValidation' })
  }
}

function validateButton(
  button: MenuButton,
  depth: number,
  seen: Set<string>,
  issues: TreeIssue[],
  workflowIds?: Set<string>
): void {
  if (!button.id || seen.has(button.id)) {
    issues.push({ key: 'menu.errDupId' })
  } else {
    seen.add(button.id)
  }
  if (!button.label.trim()) {
    issues.push({ key: 'menu.errLabel' })
  }
  validateAction(button.action, depth, seen, issues, workflowIds)
}

function validateCard(
  card: MenuCard,
  depth: number,
  seen: Set<string>,
  issues: TreeIssue[],
  workflowIds?: Set<string>
): void {
  if (!(card.text ?? '').trim() && !(card.media?.length ?? 0)) {
    issues.push({ key: 'menu.errCard' })
  }
  for (const m of card.media ?? []) {
    if (!hasHTTPPrefix(m.url)) {
      issues.push({ key: 'menu.errMedia' })
    }
  }
  for (const b of card.buttons ?? []) {
    validateButton(b, depth, seen, issues, workflowIds)
  }
}

export function validateMenuTree(
  tree: MenuTree,
  opts?: { workflowIds?: Set<string> }
): TreeIssue[] {
  const issues: TreeIssue[] = []
  if (!tree.id) {
    issues.push({ key: 'menu.saveValidation' })
  }
  if (tree.columns < 1 || tree.columns > MAX_ROOT) {
    issues.push({ key: 'menu.errColumns' })
  }
  const n = tree.items.length
  if (n < 1 || n > MAX_ROOT) {
    issues.push({ key: 'menu.errRootCount' })
  }
  const seen = new Set<string>()
  for (const item of tree.items) {
    validateButton(item, 0, seen, issues, opts?.workflowIds)
  }
  return issues
}

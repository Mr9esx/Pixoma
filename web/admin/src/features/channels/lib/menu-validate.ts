import type { MenuNode } from '@/lib/api/channel-menu'

export type MenuValidation = {
  ok: boolean
  errors: { id: string; message: string }[]
}

function isOpenCaseEmpty(node: MenuNode): boolean {
  if (node.capability_id !== 'open_case') return false
  const ids = node.params?.case_ids
  return !Array.isArray(ids) || ids.length === 0
}

function validateNode(
  node: MenuNode,
  errors: { id: string; message: string }[]
) {
  if (!node.label.trim()) errors.push({ id: node.id, message: 'name' })
  const hasChildren = (node.children?.length ?? 0) > 0
  if (!hasChildren && !node.capability_id) {
    errors.push({ id: node.id, message: 'action' })
  }
  if (isOpenCaseEmpty(node)) errors.push({ id: node.id, message: 'open_case' })
  if (hasChildren) {
    for (const child of node.children ?? []) {
      if ((child.children?.length ?? 0) > 0) {
        errors.push({ id: child.id, message: 'nested' })
      }
    }
  }
  for (const child of node.children ?? []) validateNode(child, errors)
}

export function validateMenu(items: MenuNode[]): MenuValidation {
  const errors: { id: string; message: string }[] = []
  if (items.filter((it) => it.enabled).length > 6) {
    errors.push({ id: '_root', message: 'root_count' })
  }
  for (const it of items) validateNode(it, errors)
  return { ok: errors.length === 0, errors }
}

function diffCount(a: unknown, b: unknown): number {
  return JSON.stringify(a) === JSON.stringify(b) ? 0 : 1
}

export function countUnsaved(items: MenuNode[], baseline: MenuNode[]): number {
  const byId = (list: MenuNode[]) => new Map(list.map((n) => [n.id, n]))
  const cur = byId(items)
  const base = byId(baseline)
  const ids = new Set([...cur.keys(), ...base.keys()])
  let count = 0
  for (const id of ids) {
    const a = cur.get(id)
    const b = base.get(id)
    if (!a || !b) {
      count += 1
      continue
    }
    count += diffCount(a.label, b.label)
    count += diffCount(a.capability_id, b.capability_id)
    count += diffCount(a.params, b.params)
    count += diffCount(a.intro_text, b.intro_text)
    count += diffCount(a.children, b.children)
  }
  return count
}

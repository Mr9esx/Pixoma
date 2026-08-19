import type { MenuNode } from '@/lib/api/channel-menu'

export type SimButton = {
  label: string
  kind: 'capability' | 'group'
  capabilityId?: string
}

export type SimGroup = {
  id: string
  title: string
  intro: string
  buttons: SimButton[]
}

export type TgSimulation = {
  keyboard: string[][]
  groups: Record<string, SimGroup>
}

export function rootColumnsOf(items: MenuNode[]): number {
  for (const it of items) {
    if (!it.enabled) continue
    const v = it.render_override?.columns
    if (typeof v === 'number' && v >= 1 && v <= 8) return Math.trunc(v)
  }
  return 2
}

export function buildTgSimulation(items: MenuNode[]): TgSimulation {
  const enabledRoots = items.filter((it) => it.enabled)
  const cols = rootColumnsOf(items)
  const keyboard: string[][] = []
  let row: string[] = []
  for (const it of enabledRoots) {
    row.push(it.label)
    if (row.length >= cols) {
      keyboard.push(row)
      row = []
    }
  }
  if (row.length > 0) keyboard.push(row)

  const groups: Record<string, SimGroup> = {}
  const walk = (nodes: MenuNode[]) => {
    for (const n of nodes) {
      if (n.children?.length) {
        groups[n.id] = {
          id: n.id,
          title: n.label,
          intro: n.intro_text ?? n.label,
          buttons: n.children
            .filter((c) => c.enabled)
            .map((c) => ({
              label: c.label,
              kind: c.capability_id ? 'capability' : 'group',
              capabilityId: c.capability_id,
            })),
        }
        walk(n.children)
      }
    }
  }
  walk(items)
  return { keyboard, groups }
}

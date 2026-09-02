import type { Data } from '@measured/puck'
import type { MenuAction, MenuButton, MenuCard, MenuTree } from '@/lib/api/channel-menu'

export type MenuPuckData = Data

export function uid(prefix: string): string {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}

type ButtonDraft = {
  id: string
  label: string
  actionType: string
  workflow_id: string
  text: string
  url: string
  mediaUrl: string
  cardText: string
  nestedCardText: string
  cardButtons: ButtonDraft[]
}

function draftFromButton(b: MenuButton): ButtonDraft {
  return {
    id: b.id,
    label: b.label,
    actionType: b.action.type,
    workflow_id: b.action.workflow_id ?? '',
    text: b.action.text ?? '',
    url: b.action.url ?? '',
    mediaUrl: b.action.media?.[0]?.url ?? '',
    cardText: b.action.card?.text ?? '',
    nestedCardText: b.action.card?.text ?? '',
    cardButtons: (b.action.card?.buttons ?? []).map(draftFromButton),
  }
}

function buttonToItem(b: MenuButton): Data['content'][number] {
  const d = draftFromButton(b)
  return {
    type: 'MenuButton',
    props: {
      id: d.id,
      label: d.label,
      actionType: d.actionType,
      workflow_id: d.workflow_id,
      text: d.text,
      url: d.url,
      mediaUrl: d.mediaUrl,
      cardText: d.cardText,
      cardButtons: d.cardButtons,
    },
  }
}

function asDraft(p: Record<string, unknown>): Record<string, unknown> {
  return p
}

function actionFromDraft(p: Record<string, unknown>): MenuAction {
  const type = String(p.actionType || 'send_text') as MenuAction['type']
  const mediaUrl = String(p.mediaUrl || '')
  const cardText = String(p.cardText || p.nestedCardText || '')
  const rawButtons = Array.isArray(p.cardButtons) ? p.cardButtons : []
  const card: MenuCard | undefined =
    type === 'open_card'
      ? {
          text: cardText || '卡片',
          buttons: rawButtons.map((row) => {
            const cb = asDraft((row ?? {}) as Record<string, unknown>)
            return {
              id: String(cb.id || uid('btn')),
              label: String(cb.label || '按钮'),
              action: actionFromDraft(cb),
            }
          }),
        }
      : undefined
  return {
    type,
    workflow_id: String(p.workflow_id || '') || undefined,
    text: String(p.text || '') || undefined,
    url: String(p.url || '') || undefined,
    media: mediaUrl ? [{ kind: 'image', url: mediaUrl }] : undefined,
    card,
  }
}

export function toPuckData(tree: MenuTree): MenuPuckData {
  return {
    root: { props: { columns: tree.columns || 2 } },
    content: tree.items.map(buttonToItem),
  } as MenuPuckData
}

export function fromPuckData(data: MenuPuckData, channelId: string): MenuTree {
  const columns = Number(
    (data.root as { props?: { columns?: number } })?.props?.columns ?? 2
  )
  const items: MenuButton[] = (data.content ?? []).map((item) => {
    const p = item.props as Record<string, unknown>
    return {
      id: String(p.id || uid('kbd')),
      label: String(p.label || '按钮'),
      action: actionFromDraft(p),
    }
  })
  return { id: channelId, columns: Number.isFinite(columns) ? columns : 2, items }
}

export function emptyPuckData(): MenuPuckData {
  return { root: { props: { columns: 2 } }, content: [] } as MenuPuckData
}

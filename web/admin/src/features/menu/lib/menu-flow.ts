export type ActionType =
  | 'open_card'
  | 'open_workflow'
  | 'send_text'
  | 'send_media'
  | 'open_url'
  | 'copy_text'
  | 'placeholder'

export type Action = {
  type: ActionType
  card_id?: string
  workflow_ids?: string[]
  mode?: 'list' | 'direct'
  text?: string
  media?: { kind: string; url: string; caption?: string }[]
  url?: string
}

export type MenuItem = { id: string; label: string; action: Action }
export type Menu = {
  id: string
  name: string
  columns: number
  items: MenuItem[]
}
export type CardButton = { id: string; label: string; action: Action }
export type Card = {
  id: string
  name: string
  media: { kind: string; url: string; caption?: string }[]
  text: string
  buttons: CardButton[]
}

export function buildTgFlow(menu: Menu, cards: Card[]) {
  const cols = menu.columns >= 1 && menu.columns <= 8 ? menu.columns : 2
  const keyboard: string[][] = []
  let row: string[] = []
  for (const it of menu.items) {
    row.push(it.label)
    if (row.length >= cols) {
      keyboard.push(row)
      row = []
    }
  }
  if (row.length > 0) keyboard.push(row)
  const cardById = new Map(cards.map((c) => [c.id, c]))
  return { keyboard, cardById }
}

export function validateMenuConfig(menu: Menu, cards: Card[]) {
  const errors: { path: string; message: string }[] = []
  menu.items.forEach((it, i) => {
    const path = `menu.items[${i}]`
    if (!it.label.trim()) errors.push({ path, message: 'label' })
    validateAction(it.action, path, cards, errors)
  })
  for (const card of cards) {
    card.buttons.forEach((b, i) => {
      const path = `card:${card.id}.buttons[${i}]`
      if (!b.label.trim()) errors.push({ path, message: 'label' })
      validateAction(b.action, path, cards, errors)
    })
  }
  return { ok: errors.length === 0, errors }
}

function validateAction(
  a: Action,
  path: string,
  cards: Card[],
  errors: { path: string; message: string }[]
) {
  if (a.type === 'open_card' && !a.card_id) {
    errors.push({ path, message: 'card' })
  }
  if (
    a.type === 'open_card' &&
    a.card_id &&
    !cards.some((c) => c.id === a.card_id)
  ) {
    errors.push({ path, message: 'card_missing' })
  }
  if (
    a.type === 'open_workflow' &&
    (!a.workflow_ids || a.workflow_ids.length === 0)
  ) {
    errors.push({ path, message: 'workflow' })
  }
  if (a.type === 'open_url' && !/^https?:\/\//.test(a.url ?? '')) {
    errors.push({ path, message: 'url' })
  }
  if (a.type === 'send_media' && (!a.media || a.media.length === 0)) {
    errors.push({ path, message: 'media' })
  }
}

import type {
  Action,
  ActionType,
  Card,
  CardButton,
  Menu,
  MenuItem,
} from '@/lib/api/channel-menu'
import type { WorkflowRef } from '../node-view'

export type { Action, ActionType, Card, CardButton, Menu, MenuItem }

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
  if (a.type === 'open_workflow' && !a.workflow_id) {
    errors.push({ path, message: 'workflow' })
  }
  if (a.type === 'open_url' && !/^https?:\/\//.test(a.url ?? '')) {
    errors.push({ path, message: 'url' })
  }
  if (a.type === 'send_media' && (!a.media || a.media.length === 0)) {
    errors.push({ path, message: 'media' })
  }
}

export type MapOutcomeKey =
  | 'mapOpenCard'
  | 'mapStartWorkflow'
  | 'mapSendText'
  | 'mapSendMedia'
  | 'mapOpenUrl'
  | 'mapCopyText'
  | 'mapCardMissing'
  | 'mapWorkflowMissing'

export type MapOutcome = { key: MapOutcomeKey; name?: string }

function cardById(cards: Card[], id: string | undefined): Card | undefined {
  return id ? cards.find((c) => c.id === id) : undefined
}

function workflowById(
  workflows: WorkflowRef[],
  id: string | undefined
): WorkflowRef | undefined {
  return id ? workflows.find((w) => String(w.id) === id) : undefined
}

export function actionOutcomeLabel(
  action: Action,
  cards: Card[],
  workflows: WorkflowRef[]
): MapOutcome {
  switch (action.type) {
    case 'open_card': {
      const c = cardById(cards, action.card_id)
      return c
        ? { key: 'mapOpenCard', name: c.name }
        : { key: 'mapCardMissing' }
    }
    case 'open_workflow': {
      const w = workflowById(workflows, action.workflow_id)
      return w
        ? { key: 'mapStartWorkflow', name: w.name }
        : { key: 'mapWorkflowMissing' }
    }
    case 'send_text':
      return { key: 'mapSendText' }
    case 'send_media':
      return { key: 'mapSendMedia' }
    case 'open_url':
      return { key: 'mapOpenUrl' }
    case 'copy_text':
      return { key: 'mapCopyText' }
    default:
      return { key: 'mapCardMissing' }
  }
}

export function actionIsBroken(
  action: Action,
  cards: Card[],
  workflows: WorkflowRef[]
): boolean {
  const key = actionOutcomeLabel(action, cards, workflows).key
  return key === 'mapCardMissing' || key === 'mapWorkflowMissing'
}

export function reachableCardIds(menu: Menu, cards: Card[]): Set<string> {
  const byId = new Map(cards.map((c) => [c.id, c]))
  const seen = new Set<string>()
  const walk = (action: Action) => {
    if (action.type !== 'open_card' || !action.card_id) return
    if (seen.has(action.card_id)) return
    const card = byId.get(action.card_id)
    if (!card) return
    seen.add(card.id)
    for (const b of card.buttons) walk(b.action)
  }
  for (const it of menu.items) walk(it.action)
  return seen
}

export function orphanCards(menu: Menu, cards: Card[]): Card[] {
  const reached = reachableCardIds(menu, cards)
  return cards.filter((c) => !reached.has(c.id))
}

function collectWorkflowIds(
  action: Action,
  cards: Card[],
  seen: Set<string>,
  walking: Set<string>
) {
  if (action.type === 'open_workflow' && action.workflow_id) {
    seen.add(action.workflow_id)
  }
  if (
    action.type === 'open_card' &&
    action.card_id &&
    !walking.has(action.card_id)
  ) {
    walking.add(action.card_id)
    const card = cards.find((c) => c.id === action.card_id)
    if (card)
      for (const b of card.buttons)
        collectWorkflowIds(b.action, cards, seen, walking)
  }
}

export function workflowEntryCount(menu: Menu, cards: Card[]): number {
  const seen = new Set<string>()
  const walking = new Set<string>()
  for (const it of menu.items)
    collectWorkflowIds(it.action, cards, seen, walking)
  for (const card of cards) {
    for (const b of card.buttons)
      collectWorkflowIds(b.action, cards, seen, walking)
  }
  return seen.size
}

export function mediaKindLabelKey(
  kind: string
): 'mapThumbImage' | 'mapThumbVideo' | 'mapThumbAnimation' {
  if (kind === 'video') return 'mapThumbVideo'
  if (kind === 'animation') return 'mapThumbAnimation'
  return 'mapThumbImage'
}

export type MapTrailStep = {
  kind: 'item' | 'orphan' | 'btn'
  id: string
  label: string
  action: Action
}

export function trailFromItem(item: MenuItem): MapTrailStep[] {
  return [{ kind: 'item', id: item.id, label: item.label, action: item.action }]
}

export function trailFromOrphan(card: Card): MapTrailStep[] {
  return [
    {
      kind: 'orphan',
      id: card.id,
      label: card.name,
      action: { type: 'open_card', card_id: card.id },
    },
  ]
}

export function pushTrailButton(
  trail: MapTrailStep[],
  button: CardButton
): MapTrailStep[] {
  return trail.concat([
    { kind: 'btn', id: button.id, label: button.label, action: button.action },
  ])
}

export function sliceTrail(
  trail: MapTrailStep[],
  indexInclusive: number
): MapTrailStep[] {
  return trail.slice(0, indexInclusive + 1)
}

export function clampMenuColumns(n: number): number {
  const x = Math.floor(n)
  return x >= 1 && x <= 8 ? x : 2
}

export function createBlankMenuItem(id: string): MenuItem {
  return { id, label: '', action: { type: 'send_text', text: '' } }
}

export function createBlankCardButton(id: string): CardButton {
  return { id, label: '', action: { type: 'send_text', text: '' } }
}

function stableMenu(menu: Menu): Menu {
  return {
    ...menu,
    columns: clampMenuColumns(menu.columns),
    items: menu.items.map((it) => ({ ...it })),
  }
}

function stableCards(cards: Card[]): Card[] {
  return cards.map((c) => ({
    ...c,
    media: c.media ?? [],
    buttons: c.buttons ?? [],
  }))
}

export function isMenuDraftDirty(
  saved: { menu: Menu; cards: Card[] },
  draft: { menu: Menu; cards: Card[] }
): boolean {
  return (
    JSON.stringify({
      menu: stableMenu(saved.menu),
      cards: stableCards(saved.cards),
    }) !==
    JSON.stringify({
      menu: stableMenu(draft.menu),
      cards: stableCards(draft.cards),
    })
  )
}

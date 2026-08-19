import { apiFetch } from './client'

export type MenuNode = {
  id: string
  parent_id?: string
  label: string
  order: number
  enabled: boolean
  capability_id?: string
  params?: Record<string, unknown>
  render_override?: Record<string, unknown>
  placeholder_text?: string
  intro_text?: string
  reply?: { text?: string; images?: string[] }
  children?: MenuNode[]
}

export type ChannelMenuTree = {
  channel_id: string
  items: MenuNode[]
  updated_at: string
}

export type MenuExtra = {
  channel_id: string
  menu_item_id: string
  extra_type: string
  extra_json: string
  updated_at: string
}

export function getChannelMenu(channelId: string) {
  return apiFetch<ChannelMenuTree>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu`
  )
}

export function putChannelMenu(channelId: string, items: MenuNode[]) {
  return apiFetch<ChannelMenuTree>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu`,
    { method: 'PUT', body: JSON.stringify({ items }) }
  )
}

export function getChannelMenuExtras(channelId: string) {
  return apiFetch<Record<string, MenuExtra[]>>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu/extras`
  )
}

export function putChannelMenuExtras(
  channelId: string,
  extras: Record<string, MenuExtra[]>
) {
  return apiFetch<Record<string, MenuExtra[]>>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu/extras`,
    { method: 'PUT', body: JSON.stringify(extras) }
  )
}

export type MenuPlacementPathStep = {
  id: string
  label: string
}

export type MenuPlacement = {
  channel_id: string
  item_id: string
  path: MenuPlacementPathStep[]
}

export function getCaseMenuPlacements(caseId: string) {
  return apiFetch<MenuPlacement[]>(
    `/api/v1/cases/${encodeURIComponent(caseId)}/menu-placements`
  )
}

export type PreviewDTO = {
  main_keyboard: string[][]
  groups: Record<
    string,
    { title: string; buttons: { label: string; kind: string }[] }
  >
}

export function getChannelMenuPreview(channelId: string) {
  return apiFetch<PreviewDTO>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu/preview`
  )
}

// ---- New menu/card model (replaces the legacy tree above) ----

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

export function getMenu(channelId: string) {
  return apiFetch<Menu>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu`
  )
}

export function putMenu(channelId: string, menu: Menu) {
  return apiFetch<Menu>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu`,
    { method: 'PUT', body: JSON.stringify(menu) }
  )
}

export function listCards(channelId: string, q?: string) {
  const query = q ? `?q=${encodeURIComponent(q)}` : ''
  return apiFetch<Card[]>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/cards${query}`
  )
}

export function createCard(channelId: string, card: Card) {
  return apiFetch<Card>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/cards`,
    { method: 'POST', body: JSON.stringify(card) }
  )
}

export function updateCard(channelId: string, card: Card) {
  return apiFetch<Card>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/cards/${encodeURIComponent(card.id)}`,
    { method: 'PATCH', body: JSON.stringify(card) }
  )
}

export function deleteCard(channelId: string, cardId: string) {
  return apiFetch<void>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/cards/${encodeURIComponent(cardId)}`,
    { method: 'DELETE' }
  )
}

export function getCardReferences(channelId: string, cardId: string) {
  return apiFetch<string[]>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/cards/${encodeURIComponent(cardId)}/references`
  )
}

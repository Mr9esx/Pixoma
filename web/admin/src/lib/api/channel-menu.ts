import { apiFetch } from './client'

export type MenuPlacementPathStep = {
  id: string
  label: string
}

export type MenuPlacement = {
  channel_id: string
  channel_name?: string
  item_id: string
  kind?: string
  path: MenuPlacementPathStep[]
}

export function getCaseMenuPlacements(caseId: string | number) {
  return apiFetch<MenuPlacement[]>(
    `/api/v1/cases/${encodeURIComponent(String(caseId))}/menu-placements`
  )
}

export type ActionType =
  | 'open_card'
  | 'open_workflow'
  | 'list_tasks'
  | 'send_text'
  | 'send_media'
  | 'open_url'
  | 'copy_text'

export type Media = { kind: string; url: string; caption?: string }

export type MenuAction = {
  type: ActionType
  workflow_id?: string
  text?: string
  media?: Media[]
  url?: string
  card?: MenuCard
}

export type MenuButton = { id: string; label: string; action: MenuAction }
export type MenuItem = MenuButton
export type MenuCard = {
  text: string
  media?: Media[]
  buttons?: MenuButton[]
}
export type MenuTree = {
  id: string
  columns: number
  items: MenuButton[]
}
export type Menu = MenuTree
export type Action = MenuAction

export function getMenu(channelId: string) {
  return apiFetch<MenuTree>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu`
  )
}

export function putMenu(channelId: string, menu: MenuTree) {
  return apiFetch<MenuTree>(
    `/api/v1/channels/${encodeURIComponent(channelId)}/menu`,
    { method: 'PUT', body: JSON.stringify(menu) }
  )
}
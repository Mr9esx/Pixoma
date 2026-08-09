import { apiFetch } from './client'

export type MenuKind =
  | 'folder'
  | 'open_case'
  | 'placeholder'
  | 'reply_media'
  | 'list_cases_by_tag'

export type MenuNode = {
  id: string
  parent_id?: string
  label: string
  row: number
  col: number
  enabled: boolean
  kind: MenuKind
  case_ids?: string[]
  tag?: string
  placeholder_text?: string
  intro_text?: string
  reply?: { text?: string; images?: string[] }
  children?: MenuNode[]
}

export type TgMenuTree = {
  id: string
  bot_id: string
  items: MenuNode[]
  updated_at: string
}

export function getTgMenu() {
  return apiFetch<TgMenuTree>('/api/v1/tg-menu')
}

export function putTgMenu(items: MenuNode[]) {
  return apiFetch<TgMenuTree>('/api/v1/tg-menu', {
    method: 'PUT',
    body: JSON.stringify({ items }),
  })
}

export type MenuPlacementPathStep = {
  id: string
  label: string
}

export type MenuPlacement = {
  menu_id: string
  item_id: string
  path: MenuPlacementPathStep[]
}

export function getCaseMenuPlacements(caseId: string) {
  return apiFetch<MenuPlacement[]>(
    `/api/v1/cases/${encodeURIComponent(caseId)}/menu-placements`,
  )
}

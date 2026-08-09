import { apiFetch } from './client'

export type MenuAction =
  | 'open_case'
  | 'list_cases_by_tag'
  | 'placeholder'
  | 'reply_media'

export type MenuItem = {
  id: string
  label: string
  row: number
  col: number
  enabled: boolean
  action: MenuAction
  case_id?: string
  tag?: string
  placeholder_text?: string
  reply?: { text?: string; images?: string[] }
}

export type TgMenuDocument = {
  id: string
  items: MenuItem[]
  updated_at: string
}

export function getTgMenu() {
  return apiFetch<TgMenuDocument>('/api/v1/tg-menu')
}

export function putTgMenu(items: MenuItem[]) {
  return apiFetch<TgMenuDocument>('/api/v1/tg-menu', {
    method: 'PUT',
    body: JSON.stringify({ items }),
  })
}

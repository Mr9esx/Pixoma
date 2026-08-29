import type { ActionType } from '@/lib/api/channel-menu'

/**
 * 6 种 v2 合法 action type 各自对应一个字段面板 kind。
 * - `workflow`: 单值工作流 Select（v8 收紧：单工作流）
 * - `card`: 单值卡片 Select
 * - `text`: 单 Textarea（send_text / copy_text 共用）
 * - `media`: 多 URL Textarea（每行一个媒体）
 * - `url`: 单 URL Input
 * - `none`: 不需要额外字段（占位 / 未来扩展）
 */
export type ActionFieldKind =
  | 'workflow'
  | 'card'
  | 'text'
  | 'media'
  | 'url'
  | 'none'

export const ACTION_FIELD_KIND: Record<ActionType, ActionFieldKind> = {
  open_workflow: 'workflow',
  open_card: 'card',
  send_text: 'text',
  send_media: 'media',
  open_url: 'url',
  copy_text: 'text',
}

export function getActionField(type: ActionType): ActionFieldKind {
  return ACTION_FIELD_KIND[type]
}

export function hasTemplate(type: ActionType): boolean {
  return ACTION_FIELD_KIND[type] !== 'none'
}

/**
 * v2 合法 action type 列表（顺序固定，与 i18n / Select 选项保持一致）。
 * 单一来源：增加 / 减少 type 必须同步 ActionType union 与 i18n key。
 */
export const ACTION_TYPES: readonly ActionType[] = [
  'open_workflow',
  'open_card',
  'send_text',
  'send_media',
  'open_url',
  'copy_text',
] as const

export function listActionTypes(): ActionType[] {
  return [...ACTION_TYPES]
}

/**
 * 6 个 template 字段提示文案（i18n key 前缀）。
 * ActionForm 用它生成 placeholder / helper text。
 */
export const ACTION_FIELD_LABEL_KEY: Record<ActionFieldKind, string> = {
  workflow: 'menu.workflowList',
  card: 'menu.actionOpenCard',
  text: 'menu.cardText',
  media: 'menu.cardMedia',
  url: 'menu.actionOpenUrl',
  none: '',
}

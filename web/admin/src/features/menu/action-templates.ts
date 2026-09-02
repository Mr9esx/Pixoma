import type { ActionType } from '@/lib/api/channel-menu'

/**
 * 动作类型对应的字段面板 kind。
 * - `workflow`: 单值工作流 Select
 * - `card`: 单值卡片 Select
 * - `text`: 单 Textarea（send_text / copy_text 共用）
 * - `media`: 多 URL Textarea（每行一个媒体）
 * - `url`: 单 URL Input
 * - `none`: 不需要额外字段（如 list_tasks）
 */
type ActionFieldKind = 'workflow' | 'card' | 'text' | 'media' | 'url' | 'none'

export const ACTION_FIELD_KIND: Record<ActionType, ActionFieldKind> = {
  open_workflow: 'workflow',
  list_tasks: 'none',
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
  'list_tasks',
  'open_card',
  'send_text',
  'send_media',
  'open_url',
  'copy_text',
] as const

export function listActionTypes(): ActionType[] {
  return [...ACTION_TYPES]
}

import type { Action, ActionType, Card } from '@/lib/api/channel-menu'

/** 工作流项：id + 可读名 + 卡片展示字段（来自 CaseRecord） */
export type WorkflowRef = {
  id: number
  name: string
  description?: string
  preview?: string
  tags?: string[]
  categories?: string[]
  enabled?: boolean
}

/** 从 case/工作流数据里挑出卡片展示所需的字段，两处菜单构造统一走这里。 */
export function toWorkflowRef(c: WorkflowRef): WorkflowRef {
  return {
    id: c.id,
    name: c.name,
    description: c.description,
    preview: c.preview,
    tags: c.tags,
    categories: c.categories,
    enabled: c.enabled,
  }
}

/** 结构化展示用的动作视图 */
export type ActionView = {
  type: ActionType
  /** 动作大类：workflow / card / text / media / url / copy */
  kind: 'workflow' | 'card' | 'text' | 'media' | 'url' | 'copy'
  /** 一句话摘要，如「打开工作流『写实』」 */
  summary: string
  /** 具体参数行（k/v） */
  params: { key: string; value: string }[]
}

const ACTION_TYPE_LABEL: Record<ActionType, string> = {
  open_workflow: '打开工作流',
  open_card: '打开卡片',
  send_text: '发文字',
  send_media: '发媒体',
  open_url: '跳转链接',
  copy_text: '复制文本',
}

export function actionKind(type: ActionType): ActionView['kind'] {
  switch (type) {
    case 'open_workflow':
      return 'workflow'
    case 'open_card':
      return 'card'
    case 'send_text':
    case 'copy_text':
      return 'text'
    case 'send_media':
      return 'media'
    case 'open_url':
      return 'url'
  }
}

function workflowName(id: string, workflows: WorkflowRef[]): string {
  const w = workflows.find((x) => String(x.id) === id)
  return w ? w.name : `#${id}`
}

function cardName(id: string, cards: Card[]): string {
  const c = cards.find((x) => x.id === id)
  return c && c.name ? c.name : `#${id}`
}

/** 把 action 解析成人话摘要 + 具体参数，供只读详情展示。 */
export function describeAction(
  action: Action,
  cards: Card[],
  workflows: WorkflowRef[]
): ActionView {
  const type = action.type
  const kind = actionKind(type)
  const typeLabel = ACTION_TYPE_LABEL[type]
  switch (type) {
    case 'open_workflow': {
      const name = action.workflow_id
        ? workflowName(action.workflow_id, workflows)
        : ''
      return {
        type,
        kind,
        summary: name ? `打开工作流「${name}」` : typeLabel,
        params: action.workflow_id
          ? [
              {
                key: '目标工作流',
                value: workflowName(action.workflow_id, workflows),
              },
            ]
          : [],
      }
    }
    case 'open_card': {
      const name = action.card_id ? cardName(action.card_id, cards) : ''
      return {
        type,
        kind,
        summary: name ? `打开卡片「${name}」` : typeLabel,
        params: action.card_id
          ? [{ key: '打开卡片', value: cardName(action.card_id, cards) }]
          : [],
      }
    }
    case 'send_text':
    case 'copy_text':
      return {
        type,
        kind,
        summary: action.text ? `${typeLabel}「${action.text}」` : typeLabel,
        params: action.text
          ? [{ key: '内容', value: action.text }]
          : [{ key: '内容', value: '（空）' }],
      }
    case 'send_media': {
      const urls = (action.media ?? []).filter((m) => m.url)
      return {
        type,
        kind,
        summary: urls.length > 0 ? `发媒体（${urls.length} 个）` : typeLabel,
        params: urls.map((m, i) => ({ key: `媒体 ${i + 1}`, value: m.url })),
      }
    }
    case 'open_url':
      return {
        type,
        kind,
        summary: action.url ? `跳转链接「${action.url}」` : typeLabel,
        params: action.url
          ? [{ key: '链接', value: action.url }]
          : [{ key: '链接', value: '（空）' }],
      }
  }
}

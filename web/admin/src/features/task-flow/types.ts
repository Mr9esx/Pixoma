/**
 * 真实数据结构：与 topic-routing 后端契约逐字段对齐。
 * GET /api/v1/topics、GET /api/v1/routing/attributes、
 * Case 创建/更新载荷中的 routing、GET/PATCH /api/v1/edges/{id}。
 */

export interface TopicRecord {
  key: string
  name: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface AttributeDescriptor {
  key: string
  context: 'user' | 'case' | 'input'
  label: string
  schema: {
    type?: 'string' | 'boolean' | 'number' | 'array'
    enum?: string[]
    items?: { type: string }
    description?: string
  }
}

export interface AttributesCatalogResponse {
  attributes: AttributeDescriptor[]
}

export type Condition =
  | { always: true }
  | { field: string; op: ConditionOp; value?: unknown }
  | { and: Condition[] }
  | { or: Condition[] }

export type ConditionOp =
  | 'eq'
  | 'ne'
  | 'in'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'exists'

export interface RoutingRule {
  when: Condition
  /** 目标 Topic；编辑器内允许未连线（undefined = 待配置）。 */
  topic?: string
}

export interface RoutingConfig {
  rules: RoutingRule[]
}

/** Case 文档中的路由配置（与 catalog CaseDocument.routing 对齐）。 */
export interface CaseWithRouting {
  id: number
  name: string
  routing?: RoutingConfig
}

export interface EdgeRecord {
  id: string
  name: string
  enabled: boolean
  subscribe_topics: string[]
  effective_topics: string[]
}

/** agent 心跳（GET /api/v1/edges/presence）：edge_online + comfy_running 同时为真才算在线可拉取。 */
export interface EdgePresence {
  id: string
  edge_online: boolean
  comfy_running: boolean
}

export const DEFAULT_TOPIC_KEY = 'default'

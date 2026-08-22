import { apiFetch } from './client'

export type RoutingAttributeSchema = {
  type?: 'string' | 'boolean' | 'number' | 'array'
  enum?: string[]
  items?: { type: string }
  description?: string
}

export type RoutingAttributeDescriptor = {
  key: string
  context: 'user' | 'case' | 'input'
  label: string
  schema: RoutingAttributeSchema
}

/** GET /api/v1/routing/attributes：条件属性目录（新增属性无需改前端代码）。 */
export function listRoutingAttributes() {
  return apiFetch<{ attributes: RoutingAttributeDescriptor[] }>(
    '/api/v1/routing/attributes'
  )
}

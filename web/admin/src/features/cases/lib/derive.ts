import type { InputBinding, OutputBinding } from '@/lib/api/types'
import { outputKindFor } from './node-catalog'
import type { WorkflowNode } from './workflow-parse'

export type InputFieldDraft = {
  key: string
  type: string
  required: boolean
  node_id: string
  field_path: string
  description?: string
  enum_values?: string[]
}

export type OutputFieldDraft = {
  key: string
  type: string
  node_id: string
  index?: number
  description?: string
}

function schemaTypeFor(field: InputFieldDraft): unknown {
  switch (field.type) {
    case 'number':
      return { type: 'number' }
    case 'boolean':
      return { type: 'boolean' }
    case 'enum':
      return {
        type: 'string',
        ...(field.enum_values?.length ? { enum: field.enum_values } : {}),
      }
    case 'image':
    case 'video':
      return { type: 'string' }
    default:
      return { type: 'string' }
  }
}

export function deriveInputSchema(
  inputs: InputFieldDraft[]
): Record<string, unknown> {
  const properties: Record<string, unknown> = {}
  const required: string[] = []
  for (const field of inputs) {
    const key = field.key.trim()
    if (!key) continue
    properties[key] = schemaTypeFor(field)
    if (field.required) required.push(key)
  }
  return {
    type: 'object',
    additionalProperties: false,
    required,
    properties,
  }
}

export function deriveBindings(
  inputs: InputFieldDraft[],
  outputs: OutputFieldDraft[]
): { inputs: InputBinding[]; outputs: OutputBinding[] } {
  return {
    inputs: inputs
      .filter((field) => field.key.trim() && field.node_id && field.field_path)
      .map((field) => ({
        key: field.key.trim(),
        node_id: field.node_id,
        field_path: field.field_path,
      })),
    outputs: outputs
      .filter((field) => field.key.trim() && field.node_id)
      .map((field) => ({
        key: field.key.trim(),
        node_id: field.node_id,
        index: field.index ?? 0,
      })),
  }
}

export function normalizeOutputTypes(
  outputs: OutputFieldDraft[],
  nodes: WorkflowNode[]
): OutputFieldDraft[] {
  const outputKindByNodeId = new Map(
    nodes.map((node) => [node.id, outputKindFor(node.class_type)])
  )
  return outputs.map((field) => {
    const type = outputKindByNodeId.get(field.node_id)
    return type ? { ...field, type } : field
  })
}

export function validateEditor(
  inputs: InputFieldDraft[],
  outputs: OutputFieldDraft[]
): { duplicateKey?: string; unboundInput?: string; noOutput?: boolean } {
  const out: {
    duplicateKey?: string
    unboundInput?: string
    noOutput?: boolean
  } = {}
  const seen = new Set<string>()
  for (const field of inputs) {
    const key = field.key.trim()
    if (!key) continue
    if (seen.has(key) && !out.duplicateKey) out.duplicateKey = key
    seen.add(key)
    if ((!field.node_id || !field.field_path) && !out.unboundInput) {
      out.unboundInput = key
    }
  }
  if (
    outputs.filter((field) => field.key.trim() && field.node_id).length === 0
  ) {
    out.noOutput = true
  }
  return out
}

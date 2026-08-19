import { inputKindFor, outputCountFor, type InputKind } from './node-catalog'

export type WorkflowNodeInput = { name: string; kind: InputKind }
export type WorkflowNode = {
  id: string
  class_type: string
  inputs: WorkflowNodeInput[]
  outputCount: number
}
export type WorkflowGraph = {
  nodes: WorkflowNode[]
  api: Record<string, unknown>
}
export type WorkflowParseResult =
  | { ok: true; graph: WorkflowGraph }
  | { ok: false; error: string }

type UiLink = [number, number, number, number, number, string]

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function isApiNode(v: unknown): v is Record<string, unknown> {
  return (
    isRecord(v) &&
    typeof v['class_type'] === 'string' &&
    (v['inputs'] === undefined || isRecord(v['inputs']))
  )
}

function isApiGraph(value: unknown): value is Record<string, unknown> {
  if (!isRecord(value)) return false
  const entries = Object.values(value)
  return entries.length > 0 && entries.every(isApiNode)
}

function apiInputsToNames(
  inputs: Record<string, unknown>
): WorkflowNodeInput[] {
  return Object.entries(inputs).map(([name, value]) => ({
    name,
    kind: typeof value === 'string' ? 'string' : 'unknown',
  }))
}

function convertUiToApi(
  value: Record<string, unknown>
): { ok: true; api: Record<string, unknown> } | { ok: false; error: string } {
  const nodes = value['nodes']
  if (!Array.isArray(nodes) || nodes.length === 0) {
    return { ok: false, error: 'nodes array is missing or empty' }
  }
  const linksRaw = Array.isArray(value['links']) ? value['links'] : []
  const links = new Map<number, UiLink>()
  for (const link of linksRaw) {
    if (Array.isArray(link) && typeof link[0] === 'number') {
      links.set(link[0], link as UiLink)
    }
  }
  const api: Record<string, unknown> = {}
  for (const [index, node] of nodes.entries()) {
    if (!isRecord(node)) {
      return { ok: false, error: `第 ${index + 1} 个节点格式不正确` }
    }
    const id = String(node['id'])
    const classType = node['type']
    if (!id || typeof classType !== 'string') {
      return {
        ok: false,
        error: `第 ${index + 1} 个节点缺少类型信息，无法识别`,
      }
    }
    const rawInputs = Array.isArray(node['inputs']) ? node['inputs'] : []
    const widgets = Array.isArray(node['widgets_values'])
      ? node['widgets_values']
      : []
    let widgetIndex = 0
    const inputs: Record<string, unknown> = {}
    for (const input of rawInputs) {
      if (!isRecord(input) || typeof input['name'] !== 'string') continue
      const name = input['name']
      const linkId = input['link']
      if (typeof linkId === 'number' && links.has(linkId)) {
        const link = links.get(linkId)!
        inputs[name] = [String(link[1]), link[2]]
      } else if (linkId === null && widgetIndex < widgets.length) {
        inputs[name] = widgets[widgetIndex]
        widgetIndex += 1
      }
    }
    for (; widgetIndex < widgets.length; widgetIndex += 1) {
      inputs[`widget_${widgetIndex}`] = widgets[widgetIndex]
    }
    api[id] = { class_type: classType, inputs }
  }
  return { ok: true, api }
}

export function parseWorkflow(raw: string): WorkflowParseResult {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return { ok: false, error: '不是有效的 JSON' }
  }
  if (!isRecord(parsed)) return { ok: false, error: 'JSON 顶层必须是对象' }

  let api: Record<string, unknown>
  if (isApiGraph(parsed)) {
    api = parsed
  } else if (Array.isArray(parsed['nodes'])) {
    const converted = convertUiToApi(parsed)
    if (!converted.ok) return { ok: false, error: converted.error }
    api = converted.api
  } else {
    return { ok: false, error: '不是 ComfyUI 导出的工作流' }
  }

  const nodes: WorkflowNode[] = Object.entries(api).map(([id, rawNode]) => {
    const node = rawNode as Record<string, unknown>
    const classType = String(node['class_type'])
    const inputs = isRecord(node['inputs'])
      ? apiInputsToNames(node['inputs']).map((input) => ({
          ...input,
          kind: inputKindFor(classType, input.name),
        }))
      : []
    return {
      id,
      class_type: classType,
      inputs,
      outputCount: outputCountFor(classType),
    }
  })
  return { ok: true, graph: { nodes, api } }
}

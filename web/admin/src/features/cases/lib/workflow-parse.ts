import { inputKindFor, outputCountFor, type InputKind } from './node-catalog'

export type WorkflowNodeInput = { name: string; kind: InputKind; ref: boolean }
export type WorkflowNodeLink = { name: string; src: string; slot: number }
export type WorkflowNode = {
  id: string
  class_type: string
  inputs: WorkflowNodeInput[]
  /** 字面量参数（名 → 字符串化的当前值），供绑定弹层预览。 */
  literals: Array<[string, string]>
  /** 引用参数（节点间连线），供绑定弹层拓扑排序。 */
  links: WorkflowNodeLink[]
  outputCount: number
}
export type WorkflowGraph = {
  nodes: WorkflowNode[]
  api: Record<string, unknown>
}
type WorkflowParseResult =
  | { ok: true; graph: WorkflowGraph }
  | { ok: false; error: string }

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

// hasWidgetKeys 检测图中是否残留 widget_N 占位输入——标准 API 导出不会有这种键。
function hasWidgetKeys(api: Record<string, unknown>): boolean {
  for (const raw of Object.values(api)) {
    const node = raw as Record<string, unknown>
    if (!isRecord(node)) continue
    const inputs = isRecord(node['inputs']) ? node['inputs'] : {}
    if (Object.keys(inputs).some((k) => /^widget_\d+$/.test(k))) {
      return true
    }
  }
  return false
}

export function parseWorkflow(raw: string): WorkflowParseResult {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return { ok: false, error: '不是有效的 JSON' }
  }
  if (!isRecord(parsed)) return { ok: false, error: 'JSON 顶层必须是对象' }

  if (!isApiGraph(parsed)) {
    return {
      ok: false,
      error: '不是 ComfyUI「保存(API 格式)」导出的 JSON',
    }
  }
  if (hasWidgetKeys(parsed)) {
    return {
      ok: false,
      error: '含 `widget_N` 占位输入，用「保存(API 格式)」重新导出',
    }
  }
  const api = parsed

  const nodes: WorkflowNode[] = Object.entries(api).map(([id, rawNode]) => {
    const node = rawNode as Record<string, unknown>
    const classType = String(node['class_type'])
    const rawInputs = isRecord(node['inputs']) ? node['inputs'] : {}
    const entries = Object.entries(rawInputs)
    const inputs: WorkflowNodeInput[] = entries.map(([name, value]) => {
      const ref = Array.isArray(value)
      return {
        name,
        kind: ref ? 'ref' : inputKindFor(classType, name, value),
        ref,
      }
    })
    const literals: Array<[string, string]> = []
    const links: WorkflowNodeLink[] = []
    for (const [name, value] of entries) {
      if (Array.isArray(value) && typeof value[0] === 'string') {
        links.push({
          name,
          src: value[0],
          slot: typeof value[1] === 'number' ? value[1] : 0,
        })
      } else if (value !== undefined && value !== null) {
        const s = typeof value === 'string' ? value : JSON.stringify(value)
        literals.push([name, s])
      }
    }
    return {
      id,
      class_type: classType,
      inputs,
      literals,
      links,
      outputCount: outputCountFor(classType),
    }
  })
  return { ok: true, graph: { nodes, api } }
}

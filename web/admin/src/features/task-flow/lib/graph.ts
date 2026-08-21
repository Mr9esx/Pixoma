import type { Condition, EdgePresence, EdgeRecord, RoutingConfig, RoutingRule } from '../types'
import { isEdgeOnline } from './topic-binding'

/**
 * 画布图模型：拓扑固定为
 *   Case 起始节点 → N 个条件分支节点 → N 个 Topic 目标节点
 *   Case → 默认 Topic（虚线回退，只读）
 * 规则顺序的真相源是 rules 数组（显式 order），不是画布坐标。
 */
export type NodeKind =
  | 'case-start'
  | 'condition-branch'
  | 'topic-target'
  | 'default-topic'
  | 'edge-node'

export interface FlowNodeModel {
  id: string
  kind: NodeKind
  label: string
  /** 条件分支专属：当前规则在 rules 中的下标。 */
  ruleIndex?: number
  topic?: string
  /** 计算节点专属：对应 edge 的 id。 */
  edgeId?: string
  /** 计算节点专属：agent 是否在线（edge_online && comfy_running）。 */
  online?: boolean
}

export interface FlowEdgeModel {
  id: string
  source: string
  target: string
  dashed?: boolean
}

export interface FlowGraph {
  nodes: FlowNodeModel[]
  edges: FlowEdgeModel[]
  defaultTopicKey: string
}

const BRANCH_START_Y = 80
const BRANCH_GAP = 176
const CASE_X = 0
const BRANCH_X = 300
const TOPIC_X = 920

export function describeCondition(c: Condition): string {
  if ('and' in c) return `同时满足 ${c.and.length} 个条件`
  if ('or' in c) return `满足任一 ${c.or.length} 个条件`
  const opText: Record<string, string> = {
    eq: '=',
    ne: '≠',
    in: '∈',
    gt: '>',
    gte: '≥',
    lt: '<',
    lte: '≤',
    exists: '存在',
  }
  const val =
    c.op === 'exists'
      ? ''
      : Array.isArray(c.value)
        ? `[${c.value.join(', ')}]`
        : String(c.value ?? '')
  return `${c.field} ${opText[c.op] ?? c.op}${val ? ' ' + val : ''}`
}

export function ruleLabel(rule: RoutingRule, index: number): string {
  const cond = describeCondition(rule.when)
  return `#${index + 1} ${cond}`
}

/** rules 数组 → 画布图模型（自动布局：按顺序纵向排布）。 */
export function buildGraph(routing: RoutingConfig | undefined, defaultTopicKey: string): FlowGraph {
  const rules = routing?.rules ?? []
  const nodes: FlowNodeModel[] = []
  const edges: FlowEdgeModel[] = []
  // topic key → 节点 id：同一 Topic 只渲染一个节点（规则选默认 Topic 时直接连默认节点）。
  const topicNodeIds = new Map<string, string>()

  const caseNodeId = 'case-start'
  const defaultNodeId = 'default-topic'
  nodes.push({ id: caseNodeId, kind: 'case-start', label: 'Case 任务' })
  edges.push({ id: `e-default`, source: caseNodeId, target: defaultNodeId, dashed: true })

  rules.forEach((rule, i) => {
    const branchId = `branch-${i}`
    nodes.push({
      id: branchId,
      kind: 'condition-branch',
      label: ruleLabel(rule, i),
      ruleIndex: i,
    })
    edges.push({ id: `e-case-${i}`, source: caseNodeId, target: branchId })

    if (!rule.topic) {
      // 未连线规则：无 Topic 边，等待手动连接。
      return
    }
    if (rule.topic === defaultTopicKey) {
      // 规则命中默认 Topic：与 Case 无规则回退共用同一个默认节点。
      edges.push({ id: `e-branch-${i}`, source: branchId, target: defaultNodeId })
      return
    }
    const existing = topicNodeIds.get(rule.topic)
    if (existing) {
      edges.push({ id: `e-branch-${i}`, source: branchId, target: existing })
      return
    }
    const id = `topic-${rule.topic}`
    topicNodeIds.set(rule.topic, id)
    nodes.push({ id, kind: 'topic-target', label: rule.topic, topic: rule.topic, ruleIndex: i })
    edges.push({ id: `e-branch-${i}`, source: branchId, target: id })
  })

  // 默认节点放到最后：forceNodeModelOrder 下保持在 Topic 列底部。
  nodes.push({ id: defaultNodeId, kind: 'default-topic', label: defaultTopicKey })

  return { nodes, edges, defaultTopicKey }
}

/**
 * 在路由拓扑之上叠加消费绑定层：每个「后台已绑定该 Topic」的计算节点
 * 作为独立节点，用边与 Topic 相连（edge 在线与否决定边的样式与节点状态）。
 * 只展示当前 Case 路由中出现的 Topic（含默认回退）的绑定关系。
 */
export function buildBindingGraph(
  routing: RoutingConfig | undefined,
  defaultTopicKey: string,
  edges: EdgeRecord[],
  presence: EdgePresence[],
): FlowGraph {
  const graph = buildGraph(routing, defaultTopicKey)
  const topicNodeIdByKey = new Map<string, string>()
  topicNodeIdByKey.set(defaultTopicKey, 'default-topic')
  for (const node of graph.nodes) {
    if (node.kind === 'topic-target' && node.topic) topicNodeIdByKey.set(node.topic, node.id)
  }

  const added = new Set<string>()
  for (const edge of edges) {
    if (!edge.enabled) continue
    for (const topic of edge.effective_topics) {
      const topicNodeId = topicNodeIdByKey.get(topic)
      if (!topicNodeId) continue
      const edgeNodeId = `edge-${edge.id}`
      if (!added.has(edgeNodeId)) {
        added.add(edgeNodeId)
        graph.nodes.push({ id: edgeNodeId, kind: 'edge-node', label: edge.name, edgeId: edge.id, online: isEdgeOnline(presence, edge.id) })
      }
      graph.edges.push({
        id: `e-bind-${edge.id}-${topic}`,
        source: topicNodeId,
        target: edgeNodeId,
        // 离线 agent 的绑定关系用虚线弱化：绑定成立，但当前拉取不可用。
        dashed: !isEdgeOnline(presence, edge.id),
      })
    }
  }
  return graph
}

/** 节点纵向位置（React Flow 使用）。顺序由 rules 数组决定。 */
export function nodePosition(node: FlowNodeModel, totalRules: number): { x: number; y: number } {
  const x =
    node.kind === 'case-start'
      ? CASE_X
      : node.kind === 'condition-branch'
        ? BRANCH_X
        : TOPIC_X
  if (node.kind === 'case-start') {
    const top = totalRules > 0 ? BRANCH_START_Y : BRANCH_START_Y + 80
    const span = totalRules > 0 ? (totalRules - 1) * BRANCH_GAP : 0
    return { x, y: top + span / 2 }
  }
  if (node.kind === 'default-topic') {
    return { x: 640, y: totalRules > 0 ? BRANCH_START_Y + totalRules * BRANCH_GAP : BRANCH_START_Y + 120 }
  }
  const index = node.ruleIndex ?? 0
  return { x, y: BRANCH_START_Y + index * BRANCH_GAP }
}

/** 按画布拖拽后的 y 坐标重排规则（稳定排序，未拖动的保持相对顺序）。 */
export function reorderByY(routing: RoutingConfig, positions: { id: string; y: number }[]): RoutingConfig {
  const byId = new Map(positions.map((p) => [p.id, p.y]))
  const indexed = routing.rules.map((rule, i) => ({ rule, i, y: byId.get(`branch-${i}`) ?? i * BRANCH_GAP }))
  indexed.sort((a, b) => a.y - b.y)
  return { rules: indexed.map((x) => x.rule) }
}

/** 分支上移/下移（调整 rules 顺序并重建图）。 */
export function moveRule(routing: RoutingConfig, index: number, dir: -1 | 1): RoutingConfig {
  const target = index + dir
  if (index < 0 || target < 0 || target >= routing.rules.length) return routing
  const rules = [...routing.rules]
  const [item] = rules.splice(index, 1)
  rules.splice(target, 0, item)
  return { rules }
}

/** 追加/删除规则。 */
export function addRule(routing: RoutingConfig | undefined): RoutingConfig {
  const rules = [...(routing?.rules ?? [])]
  rules.push({
    when: { field: 'user.is_premium', op: 'eq', value: true },
    topic: undefined, // 新规则默认未连线，由用户手动拖线到 Topic
  })
  return { rules }
}

export function removeRule(routing: RoutingConfig, index: number): RoutingConfig {
  const rules = routing.rules.filter((_, i) => i !== index)
  return { rules }
}

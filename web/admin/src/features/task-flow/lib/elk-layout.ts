import ELK from 'elkjs/lib/elk.bundled.js'
import type { ElkNode } from 'elkjs/lib/elk-api'
import type { Edge, Node } from '@xyflow/react'

/**
 * ELK 自动布局（对齐官方 reactflow.dev/examples/layout/elkjs-multiple-handles）：
 * - layered + RIGHT 方向：Case 在最左、条件分支居中、Topic 在右侧。
 * - 每个节点声明 ports（target=WEST / source=EAST），配合 FIXED_ORDER
 *   让规则顺序即端口顺序，拖拽排序后边不会交叉。
 * - 默认 Topic 用 layerConstraint=LAST 固定到最右的 Topic 列。
 */

const elk = new ELK()

export const ELK_LAYOUT_OPTIONS = {
  'elk.algorithm': 'layered',
  'elk.direction': 'RIGHT',
  'elk.layered.spacing.edgeNodeBetweenLayers': '60',
  'elk.spacing.nodeNode': '60',
  'elk.layered.nodePlacement.strategy': 'SIMPLE',
  // 保持输入顺序（规则数组顺序），避免单出口扇出时被交叉最小化打乱。
  'elk.layered.crossingMinimization.forceNodeModelOrder': 'true',
} as const

export const CASE_NODE_ID = 'case-start'
export const DEFAULT_NODE_ID = 'default-topic'
export const DEFAULT_TARGET_HANDLE = 'default-t'
export const DEFAULT_SOURCE_HANDLE = 'default-s'
export const CASE_SOURCE_HANDLE = 'case-s'

export function branchTargetHandle(index: number): string {
  return `branch-${index}-t`
}

export function branchSourceHandle(index: number): string {
  return `branch-${index}-s`
}

export function topicTargetHandle(topicKey: string): string {
  return `topic-${topicKey}-t`
}

export function topicSourceHandle(topicKey: string): string {
  return `topic-${topicKey}-s`
}

export function edgeTargetHandle(edgeId: string): string {
  return `edge-${edgeId}-t`
}

type HandleData = {
  sourceHandles?: string[]
  targetHandles?: string[]
}

/**
 * 纯布局函数：把 React Flow 节点/边交给 ELK，返回带新 position 的节点。
 * 节点 data 里需要 sourceHandles/targetHandles（画布 buildNodes 时写入）。
 */
export async function elkLayout(nodes: Node[], edges: Edge[]): Promise<Node[]> {
  // 计算节点不参与 ELK 图谱：布局完成后按绑定 Topic 对齐到最右列。
  const edgeNodeIds = new Set(nodes.filter((n) => n.type === 'edge-node').map((n) => n.id))
  const layoutable = nodes.filter((n) => !edgeNodeIds.has(n.id))
  const layoutableEdges = edges.filter((e) => !edgeNodeIds.has(e.target))

  const graph: ElkNode = {
    id: 'root',
    layoutOptions: ELK_LAYOUT_OPTIONS,
    children: layoutable.map((node) => {
      const data = (node.data ?? {}) as HandleData
      const targets = (data.targetHandles ?? []).map((id) => ({ id, properties: { side: 'WEST' } }))
      const sources = (data.sourceHandles ?? []).map((id) => ({ id, properties: { side: 'EAST' } }))
      const isDefaultTopic = node.type === 'default-topic'
      return {
        id: node.id,
        width: node.measured?.width ?? 180,
        height: node.measured?.height ?? 60,
        properties: {
          'org.eclipse.elk.portConstraints': 'FIXED_ORDER',
          // 默认 Topic 属于 Topic 列：强制进入最右层，避免被 ELK 排到中间规则层。
          ...(isDefaultTopic ? { 'org.eclipse.elk.layered.layering.layerConstraint': 'LAST' } : {}),
        },
        // 官方示例做法：先放一个以节点 id 命名的空端口，兜底未显式指定 handle 的边。
        ports: [{ id: node.id }, ...targets, ...sources],
      }
    }),
    edges: layoutableEdges.map((edge) => ({
      id: edge.id,
      sources: [edge.sourceHandle ?? edge.source],
      targets: [edge.targetHandle ?? edge.target],
    })),
  }

  const layoutedGraph = await elk.layout(graph)
  const byId = new Map(layoutedGraph.children?.map((n) => [n.id, n]))

  const bindingByEdge = new Map<string, string[]>()
  for (const edge of edges) {
    if (!edgeNodeIds.has(edge.target)) continue
    const topics = bindingByEdge.get(edge.target) ?? []
    topics.push(edge.source)
    bindingByEdge.set(edge.target, topics)
  }

  return nodes.map((node) => {
    if (edgeNodeIds.has(node.id)) {
      const boundTopicId = bindingByEdge.get(node.id)?.[0]
      const topic = boundTopicId ? byId.get(boundTopicId) : undefined
      if (topic && topic.x !== undefined && topic.y !== undefined) {
        const topicWidth = topic.width ?? 200
        const topicHeight = topic.height ?? 40
        const nodeHeight = node.measured?.height ?? 44
        return {
          ...node,
          position: {
            x: topic.x + topicWidth + 80,
            y: topic.y + (topicHeight - nodeHeight) / 2,
          },
        }
      }
      return node
    }
    const layouted = byId.get(node.id)
    return {
      ...node,
      position: { x: layouted?.x ?? node.position.x, y: layouted?.y ?? node.position.y },
    }
  })
}

/**
 * 由布局结果计算适配视口的 viewport（padding 与 React Flow 语义一致：
 * 0.25 = 上下左右各留 25% 视口空间）。
 */
export function computeFitViewport(
  nodes: Node[],
  width: number,
  height: number,
  padding = 0.08,
  maxZoom = 1,
): { x: number; y: number; zoom: number } {
  if (nodes.length === 0 || width <= 0 || height <= 0) return { x: 0, y: 0, zoom: 1 }
  const xs: number[] = []
  const ys: number[] = []
  for (const n of nodes) {
    const w = n.measured?.width ?? 180
    const h = n.measured?.height ?? 60
    xs.push(n.position.x, n.position.x + w)
    ys.push(n.position.y, n.position.y + h)
  }
  const minX = Math.min(...xs)
  const maxX = Math.max(...xs)
  const minY = Math.min(...ys)
  const maxY = Math.max(...ys)
  const boundsWidth = Math.max(maxX - minX, 1)
  const boundsHeight = Math.max(maxY - minY, 1)
  const zoom = Math.min((width * (1 - 2 * padding)) / boundsWidth, (height * (1 - 2 * padding)) / boundsHeight, maxZoom)
  const centerX = (minX + maxX) / 2
  const centerY = (minY + maxY) / 2
  return { x: width / 2 - centerX * zoom, y: height / 2 - centerY * zoom, zoom }
}

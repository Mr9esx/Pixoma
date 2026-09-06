import type { LinkHealthGraph } from '@/lib/api/link-health'

export type TopologyKind = 'platform' | 'case' | 'topic' | 'edge'
export type NodeHealth = 'ok' | 'warn' | 'pending'

export type GraphNode = {
  id: string
  kind: TopologyKind
  name: string
  to: string
  health: NodeHealth
}

export type GraphEdge = { id: string; source: string; target: string }

export type TopologyGraph = { nodes: GraphNode[]; edges: GraphEdge[] }

export type GraphMode =
  | { type: 'all' }
  | { type: 'focus'; kind: TopologyKind; id: string }

export function nodeId(kind: TopologyKind, id: string): string {
  return `${kind}:${id}`
}

export function parseNodeId(
  id: string
): { kind: TopologyKind; id: string } | null {
  const i = id.indexOf(':')
  if (i <= 0) return null
  return { kind: id.slice(0, i) as TopologyKind, id: id.slice(i + 1) }
}

function displayHealth(state: string): NodeHealth {
  if (state === 'ok') return 'ok'
  if (state === 'pending') return 'pending'
  return 'warn'
}

export function buildLinkGraph(
  graph: LinkHealthGraph,
  mode: GraphMode
): TopologyGraph {
  const rawNodes: GraphNode[] = graph.nodes.map((n) => ({
    id: n.id,
    kind: n.kind,
    name: n.name,
    to: n.kind === 'case' ? `/cases/${n.ref_id}` : kindPath(n.kind, n.ref_id),
    health: displayHealth(n.health),
  }))
  const rawEdges: GraphEdge[] = graph.edges.map((e) => ({
    id: `${e.from}->${e.to}`,
    source: e.from,
    target: e.to,
  }))

  const nodeIds = new Set(rawNodes.map((n) => n.id))
  const connectedEdges = rawEdges.filter(
    (e) => nodeIds.has(e.source) && nodeIds.has(e.target)
  )
  const degree = new Map<string, number>()
  for (const n of rawNodes) degree.set(n.id, 0)
  for (const e of connectedEdges) {
    degree.set(e.source, (degree.get(e.source) ?? 0) + 1)
    degree.set(e.target, (degree.get(e.target) ?? 0) + 1)
  }

  if (mode.type === 'all') {
    const keep = new Set(
      rawNodes.filter((n) => (degree.get(n.id) ?? 0) > 0).map((n) => n.id)
    )
    return {
      nodes: rawNodes.filter((n) => keep.has(n.id)),
      edges: connectedEdges.filter(
        (e) => keep.has(e.source) && keep.has(e.target)
      ),
    }
  }

  const focus = nodeId(mode.kind, mode.id)
  const byId = new Map(rawNodes.map((n) => [n.id, n]))
  const focusNode = byId.get(focus)
  if (!focusNode) {
    return { nodes: [], edges: [] }
  }
  const incoming = new Map<string, string[]>()
  const outgoing = new Map<string, string[]>()
  for (const e of connectedEdges) {
    outgoing.set(e.source, [...(outgoing.get(e.source) ?? []), e.target])
    incoming.set(e.target, [...(incoming.get(e.target) ?? []), e.source])
  }
  const walk = (start: string, map: Map<string, string[]>) => {
    const seen = new Set<string>()
    const stack = [start]
    while (stack.length) {
      const cur = stack.pop()!
      for (const next of map.get(cur) ?? []) {
        if (seen.has(next)) continue
        seen.add(next)
        stack.push(next)
      }
    }
    return seen
  }
  const keep = new Set<string>([
    focus,
    ...walk(focus, incoming),
    ...walk(focus, outgoing),
  ])
  return {
    nodes: rawNodes.filter((n) => keep.has(n.id)),
    edges: connectedEdges.filter((e) => keep.has(e.source) && keep.has(e.target)),
  }
}

function kindPath(kind: TopologyKind, refId: string): string {
  const enc = encodeURIComponent(refId)
  switch (kind) {
    case 'platform':
      return `/channels/${enc}`
    case 'topic':
      return `/topics/${enc}`
    default:
      return `/edges/${enc}`
  }
}

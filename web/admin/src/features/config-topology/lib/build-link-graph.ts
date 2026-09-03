import type { MenuTree } from '@/lib/api/channel-menu'
import type { ChannelReachability } from '@/lib/api/channels'
import type { EdgePresence, RoutingConfig } from '@/features/task-flow/types'
import {
  caseReferences,
  caseRoutingTopics,
  channelReferences,
  edgeReferences,
  edgeTopics,
  topicReferences,
} from '../../link-health/lib/references'
import { walkMenuWorkflows } from './walk-menu-workflows'

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

export type TopologySource = {
  channels: { id: string; name: string }[]
  menus: Record<string, MenuTree | undefined>
  cases: { id: number; name: string; routing?: RoutingConfig }[]
  topics: { key: string; name: string }[]
  edges: {
    id: string
    name: string
    enabled: boolean
    subscribe_topics?: string[]
    effective_topics?: string[]
  }[]
  presence: EdgePresence[]
  reachability: Record<string, ChannelReachability | undefined>
}

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

function toDisplayHealth(
  state: 'ok' | 'warn' | 'bad'
): Exclude<NodeHealth, 'pending'> {
  return state === 'ok' ? 'ok' : 'warn'
}

export function buildLinkGraph(
  input: TopologySource,
  mode: GraphMode
): TopologyGraph {
  const healthInput = {
    cases: input.cases,
    edges: input.edges,
    presence: input.presence,
  }

  const rawNodes: GraphNode[] = []
  const rawEdges: GraphEdge[] = []
  const edgeKeys = new Set<string>()

  const addEdge = (source: string, target: string) => {
    const id = `${source}->${target}`
    if (edgeKeys.has(id)) return
    edgeKeys.add(id)
    rawEdges.push({ id, source, target })
  }

  for (const ch of input.channels) {
    const reach = input.reachability[ch.id]
    rawNodes.push({
      id: nodeId('platform', ch.id),
      kind: 'platform',
      name: ch.name,
      to: `/channels/${encodeURIComponent(ch.id)}`,
      health: reach ? toDisplayHealth(channelReferences(ch.id, reach).state) : 'pending',
    })
    const workflowIds = walkMenuWorkflows(
      input.menus[ch.id] ?? { id: '', columns: 1, items: [] }
    )
    for (const wf of workflowIds) {
      addEdge(nodeId('platform', ch.id), nodeId('case', wf))
    }
  }

  for (const c of input.cases) {
    const cid = String(c.id)
    const placements = input.channels.flatMap((ch) => {
      const ids = walkMenuWorkflows(
        input.menus[ch.id] ?? { id: '', columns: 1, items: [] }
      )
      if (!ids.includes(cid)) return []
      return [
        {
          channel_id: ch.id,
          channel_name: ch.name,
          item_id: ch.id,
          path: [{ id: ch.id, label: ch.name }],
        },
      ]
    })
    const health = caseReferences(c.id, { ...healthInput, placements })
    rawNodes.push({
      id: nodeId('case', cid),
      kind: 'case',
      name: c.name,
      to: `/cases/${cid}`,
      health: toDisplayHealth(health.health.state),
    })
    for (const topic of caseRoutingTopics(c.routing)) {
      addEdge(nodeId('case', cid), nodeId('topic', topic))
    }
  }

  const topicKeys = new Set(input.topics.map((t) => t.key))
  for (const e of input.edges) {
    for (const k of edgeTopics(e)) topicKeys.add(k)
  }
  for (const c of input.cases) {
    for (const k of caseRoutingTopics(c.routing)) topicKeys.add(k)
  }

  for (const key of topicKeys) {
    const named = input.topics.find((t) => t.key === key)
    const health = topicReferences(key, healthInput)
    rawNodes.push({
      id: nodeId('topic', key),
      kind: 'topic',
      name: named?.name ?? key,
      to: `/topics/${encodeURIComponent(key)}`,
      health: toDisplayHealth(health.health.state),
    })
  }

  for (const e of input.edges) {
    const health = edgeReferences(e.id, healthInput)
    rawNodes.push({
      id: nodeId('edge', e.id),
      kind: 'edge',
      name: e.name,
      to: `/edges/${encodeURIComponent(e.id)}`,
      health: toDisplayHealth(health.health.state),
    })
    for (const k of edgeTopics(e)) {
      addEdge(nodeId('topic', k), nodeId('edge', e.id))
    }
  }

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

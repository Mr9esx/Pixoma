import type { TopologyGraph } from './build-link-graph'

export function pathThrough(
  graph: TopologyGraph,
  nodeId: string
): { nodes: Set<string>; edges: Set<string> } {
  const incoming = new Map<string, string[]>()
  const outgoing = new Map<string, string[]>()
  for (const e of graph.edges) {
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
  const nodes = new Set<string>([
    nodeId,
    ...walk(nodeId, incoming),
    ...walk(nodeId, outgoing),
  ])
  const edges = new Set(
    graph.edges
      .filter((e) => nodes.has(e.source) && nodes.has(e.target))
      .map((e) => e.id)
  )
  return { nodes, edges }
}

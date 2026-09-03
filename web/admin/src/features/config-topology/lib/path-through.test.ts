import { describe, expect, it } from 'vitest'
import { pathThrough } from './path-through'
import type { TopologyGraph } from './build-link-graph'

const graph: TopologyGraph = {
  nodes: [
    { id: 'platform:ch1', kind: 'platform', name: 'TG', to: '/channels/ch1', health: 'ok' },
    { id: 'case:1', kind: 'case', name: 'C', to: '/cases/1', health: 'ok' },
    { id: 'case:2', kind: 'case', name: 'Other', to: '/cases/2', health: 'ok' },
    { id: 'topic:t1', kind: 'topic', name: 'T', to: '/topics/t1', health: 'ok' },
    { id: 'edge:n1', kind: 'edge', name: 'N', to: '/edges/n1', health: 'ok' },
  ],
  edges: [
    { id: 'e1', source: 'platform:ch1', target: 'case:1' },
    { id: 'e2', source: 'case:1', target: 'topic:t1' },
    { id: 'e3', source: 'topic:t1', target: 'edge:n1' },
    { id: 'e4', source: 'platform:ch1', target: 'case:2' },
  ],
}

describe('pathThrough', () => {
  it('highlights ancestors and descendants of a topic', () => {
    const hit = pathThrough(graph, 'topic:t1')
    expect([...hit.nodes].sort()).toEqual([
      'case:1',
      'edge:n1',
      'platform:ch1',
      'topic:t1',
    ])
    expect(hit.nodes.has('case:2')).toBe(false)
    expect(hit.edges.has('e4')).toBe(false)
    expect(hit.edges.has('e2')).toBe(true)
  })
})

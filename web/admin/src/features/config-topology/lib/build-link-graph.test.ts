import { describe, expect, it } from 'vitest'
import type { LinkHealthGraph, LinkHealthNode } from '@/lib/api/link-health'
import { buildLinkGraph } from './build-link-graph'

function node(
  id: string,
  kind: LinkHealthNode['kind'],
  health: LinkHealthNode['health'] = 'ok'
): LinkHealthNode {
  const ref_id = id.slice(id.indexOf(':') + 1)
  return {
    id,
    kind,
    ref_id,
    name: id,
    health,
    breakpoints: [],
    upstream: [],
    downstream: [],
  }
}

const connected: LinkHealthGraph = {
  nodes: [
    node('platform:ch1', 'platform'),
    node('case:1', 'case'),
    node('topic:t1', 'topic'),
    node('edge:n1', 'edge'),
  ],
  edges: [
    { from: 'platform:ch1', to: 'case:1' },
    { from: 'case:1', to: 'topic:t1' },
    { from: 'topic:t1', to: 'edge:n1' },
  ],
}

describe('buildLinkGraph all', () => {
  it('keeps connected four-layer nodes and edges', () => {
    const graph = buildLinkGraph(connected, { type: 'all' })
    expect(graph.nodes.map((n) => n.id).sort()).toEqual([
      'case:1',
      'edge:n1',
      'platform:ch1',
      'topic:t1',
    ])
    expect(graph.edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ source: 'case:1', target: 'topic:t1' }),
        expect.objectContaining({ source: 'topic:t1', target: 'edge:n1' }),
      ])
    )
  })

  it('drops isolated resources from the full graph', () => {
    const graph = buildLinkGraph(
      {
        nodes: [...connected.nodes, node('case:99', 'case')],
        edges: connected.edges,
      },
      { type: 'all' }
    )
    expect(graph.nodes.some((n) => n.id === 'case:99')).toBe(false)
  })

  it('keeps a case with downstream even without a menu entry', () => {
    const graph = buildLinkGraph(
      {
        nodes: [
          node('platform:ch1', 'platform'),
          node('case:1', 'case'),
          node('topic:t1', 'topic'),
          node('edge:n1', 'edge'),
        ],
        edges: [
          { from: 'case:1', to: 'topic:t1' },
          { from: 'topic:t1', to: 'edge:n1' },
        ],
      },
      { type: 'all' }
    )
    expect(graph.nodes.some((n) => n.id === 'case:1')).toBe(true)
    expect(graph.nodes.some((n) => n.id === 'platform:ch1')).toBe(false)
  })

  it('passes through pending platform health', () => {
    const graph = buildLinkGraph(
      {
        ...connected,
        nodes: connected.nodes.map((n) =>
          n.id === 'platform:ch1' ? { ...n, health: 'pending' } : n
        ),
      },
      { type: 'all' }
    )
    expect(graph.nodes.find((n) => n.id === 'platform:ch1')?.health).toBe(
      'pending'
    )
  })

  it('drops a platform with no remaining edges', () => {
    const graph = buildLinkGraph(
      { nodes: [node('platform:ch1', 'platform')], edges: [] },
      { type: 'all' }
    )
    expect(graph.nodes).toEqual([])
    expect(graph.edges).toEqual([])
  })

  it('passes through warn node health', () => {
    const graph = buildLinkGraph(
      {
        ...connected,
        nodes: connected.nodes.map((n) =>
          n.id === 'edge:n1' ? { ...n, health: 'warn' } : n
        ),
      },
      { type: 'all' }
    )
    expect(graph.nodes.find((n) => n.id === 'edge:n1')?.health).toBe('warn')
  })
})

describe('buildLinkGraph focus', () => {
  it('keeps only the focused node when it has no edges', () => {
    const graph = buildLinkGraph(
      {
        nodes: [node('case:7', 'case')],
        edges: [],
      },
      { type: 'focus', kind: 'case', id: '7' }
    )
    expect(graph.nodes).toEqual([
      expect.objectContaining({ id: 'case:7', kind: 'case' }),
    ])
    expect(graph.edges).toEqual([])
  })
})

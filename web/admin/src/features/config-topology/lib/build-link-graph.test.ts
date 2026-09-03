import { describe, expect, it } from 'vitest'
import type { MenuTree } from '@/lib/api/channel-menu'
import type { ChannelReachability } from '@/lib/api/channels'
import type { EdgePresence } from '@/features/task-flow/types'
import { buildLinkGraph, type TopologySource } from './build-link-graph'

const menuOpen = (workflowId: string): MenuTree => ({
  id: 'm',
  columns: 1,
  items: [
    {
      id: 'b',
      label: '开',
      action: { type: 'open_workflow', workflow_id: workflowId },
    },
  ],
})

const source = (over: Partial<TopologySource> = {}): TopologySource => ({
  channels: [{ id: 'ch1', name: 'TG' }],
  menus: { ch1: menuOpen('1') },
  cases: [
    {
      id: 1,
      name: '工作流一',
      routing: { rules: [{ when: { always: true }, topic: 't1' }] },
    },
  ],
  topics: [{ key: 't1', name: '队列一' }],
  edges: [
    {
      id: 'n1',
      name: '节点一',
      enabled: true,
      subscribe_topics: ['t1'],
      effective_topics: ['t1'],
    },
  ],
  presence: [
    { id: 'n1', edge_online: true, comfy_running: true },
  ] satisfies EdgePresence[],
  reachability: {
    ch1: { ok: true, kind: 'ok', message: '' } satisfies ChannelReachability,
  },
  ...over,
})

describe('buildLinkGraph all', () => {
  it('builds four-layer edges and merges duplicate platform-case links', () => {
    const menu: MenuTree = {
      id: 'm',
      columns: 1,
      items: [
        {
          id: 'a',
          label: 'A',
          action: { type: 'open_workflow', workflow_id: '1' },
        },
        {
          id: 'b',
          label: 'B',
          action: { type: 'open_workflow', workflow_id: '1' },
        },
      ],
    }
    const graph = buildLinkGraph(source({ menus: { ch1: menu } }), {
      type: 'all',
    })
    expect(graph.nodes.map((n) => n.id).sort()).toEqual([
      'case:1',
      'edge:n1',
      'platform:ch1',
      'topic:t1',
    ])
    const pc = graph.edges.filter(
      (e) => e.source === 'platform:ch1' && e.target === 'case:1'
    )
    expect(pc).toHaveLength(1)
    expect(graph.edges).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ source: 'case:1', target: 'topic:t1' }),
        expect.objectContaining({ source: 'topic:t1', target: 'edge:n1' }),
      ])
    )
  })

  it('drops isolated resources from the full graph', () => {
    const graph = buildLinkGraph(
      source({
        cases: [
          {
            id: 1,
            name: '工作流一',
            routing: { rules: [{ when: { always: true }, topic: 't1' }] },
          },
          { id: 99, name: '孤立', routing: { rules: [] } },
        ],
      }),
      { type: 'all' }
    )
    expect(graph.nodes.some((n) => n.id === 'case:99')).toBe(false)
  })

  it('keeps a case with downstream even without a menu entry', () => {
    const graph = buildLinkGraph(
      source({
        menus: {
          ch1: { id: 'm', columns: 1, items: [] },
        },
      }),
      { type: 'all' }
    )
    expect(graph.nodes.some((n) => n.id === 'case:1')).toBe(true)
    expect(graph.nodes.some((n) => n.id === 'platform:ch1')).toBe(false)
  })

  it('marks platform health pending when reachability is missing', () => {
    const graph = buildLinkGraph(source({ reachability: {} }), { type: 'all' })
    expect(graph.nodes.find((n) => n.id === 'platform:ch1')?.health).toBe(
      'pending'
    )
  })

  it('drops a platform whose only menu target is a missing workflow', () => {
    const graph = buildLinkGraph(
      source({
        menus: { ch1: menuOpen('missing') },
        cases: [],
        topics: [],
        edges: [],
      }),
      { type: 'all' }
    )
    expect(graph.nodes).toEqual([])
    expect(graph.edges).toEqual([])
  })

  it('marks an offline node warn using edgeReferences', () => {
    const graph = buildLinkGraph(
      source({
        presence: [{ id: 'n1', edge_online: false, comfy_running: false }],
      }),
      { type: 'all' }
    )
    expect(graph.nodes.find((n) => n.id === 'edge:n1')?.health).toBe('warn')
  })
})

describe('buildLinkGraph focus', () => {
  it('keeps only the focused node when it has no edges', () => {
    const graph = buildLinkGraph(
      source({
        cases: [{ id: 7, name: '空', routing: { rules: [] } }],
        menus: { ch1: { id: 'm', columns: 1, items: [] } },
        topics: [],
        edges: [],
      }),
      { type: 'focus', kind: 'case', id: '7' }
    )
    expect(graph.nodes).toEqual([
      expect.objectContaining({ id: 'case:7', kind: 'case' }),
    ])
    expect(graph.edges).toEqual([])
  })
})

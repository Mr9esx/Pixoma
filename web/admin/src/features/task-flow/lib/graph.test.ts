import { describe, expect, it } from 'vitest'
import { mockRouting } from '../mock-data'
import type { EdgePresence, EdgeRecord } from '../types'
import { addRule, buildBindingGraph, buildGraph, describeCondition, moveRule, removeRule } from './graph'

const edges: EdgeRecord[] = [
  { id: 'gpu-a', name: '客厅 4090', enabled: true, subscribe_topics: ['default'], effective_topics: ['default'] },
  { id: 'gpu-b', name: '工作室双卡', enabled: true, subscribe_topics: ['fast-gpu'], effective_topics: ['fast-gpu'] },
  { id: 'gpu-c', name: '备用夜机', enabled: true, subscribe_topics: ['batch-night'], effective_topics: ['batch-night'] },
]

const presence: EdgePresence[] = [
  { id: 'gpu-a', edge_online: true, comfy_running: true },
  { id: 'gpu-b', edge_online: true, comfy_running: true },
  { id: 'gpu-c', edge_online: false, comfy_running: false },
]

describe('buildGraph', () => {
  it('把 rules 映射为 Case→分支→Topic 固定拓扑，并保留默认回退虚线边', () => {
    const g = buildGraph(mockRouting, 'default')
    expect(g.nodes.filter((n) => n.kind === 'case-start')).toHaveLength(1)
    expect(g.nodes.filter((n) => n.kind === 'condition-branch')).toHaveLength(3)
    // 规则命中 default 时与回退共用默认节点，因此 topic-target 只有 fast-gpu / batch-night 两个。
    expect(g.nodes.filter((n) => n.kind === 'topic-target')).toHaveLength(2)
    expect(g.nodes.find((n) => n.kind === 'default-topic')?.label).toBe('default')
    expect(g.edges.find((e) => e.dashed)?.target).toBe('default-topic')
    expect(g.edges).toHaveLength(3 * 2 + 1)
  })

  it('同一 Topic 只渲染一个节点；规则选默认 Topic 时直连默认节点', () => {
    const g = buildGraph(
      {
        rules: [
          { when: { field: 'user.is_premium', op: 'eq', value: true }, topic: 'fast-gpu' },
          { when: { field: 'case.category', op: 'eq', value: 'image' }, topic: 'default' },
          { when: { field: 'case.tags', op: 'in', value: ['4k'] }, topic: 'fast-gpu' },
        ],
      },
      'default',
    )
    expect(g.nodes.filter((n) => n.kind === 'topic-target')).toHaveLength(1)
    expect(g.edges.filter((e) => e.target === 'default-topic')).toHaveLength(2) // 回退 + 规则直连
    expect(g.edges.filter((e) => e.target === 'topic-fast-gpu')).toHaveLength(2)
  })

  it('空配置只显示起始节点与默认回退', () => {
    const g = buildGraph(undefined, 'default')
    expect(g.nodes.filter((n) => n.kind === 'condition-branch')).toHaveLength(0)
    expect(g.edges.filter((e) => e.dashed)).toHaveLength(1)
  })

  it('计算节点作为独立 Node 与 Topic 连线；仅展示当前 Case 路由内 Topic 的绑定', () => {
    const g = buildBindingGraph(mockRouting, 'default', edges, presence)
    expect(g.nodes.filter((n) => n.kind === 'edge-node')).toHaveLength(3) // default/fast-gpu/batch-night 各一
    expect(g.nodes.map((n) => n.id)).toContain('edge-gpu-a')
    expect(g.edges.find((e) => e.id === 'e-bind-gpu-a-default')).toMatchObject({
      source: 'default-topic',
      target: 'edge-gpu-a',
      dashed: false,
    })
    // 离线的 gpu-c 绑定边为虚线（绑定成立但当前不可拉取）
    expect(g.edges.find((e) => e.id === 'e-bind-gpu-c-batch-night')?.dashed).toBe(true)
    // cpu-low 未绑定任何 edge，不产生节点
    expect(g.nodes.some((n) => n.id === 'edge-cpu-low')).toBe(false)
  })
})

describe('describeCondition', () => {
  it('渲染叶子与组合的人类可读文本', () => {
    expect(describeCondition({ field: 'user.is_premium', op: 'eq', value: true })).toBe(
      'user.is_premium = true',
    )
    expect(describeCondition({ field: 'case.category', op: 'in', value: ['image'] })).toBe(
      'case.category ∈ [image]',
    )
    expect(describeCondition({ or: [{ field: 'a', op: 'exists' }] })).toBe('满足任一 1 个条件')
  })
})

describe('rule order operations', () => {
  it('moveRule 交换顺序并保持不可变', () => {
    const moved = moveRule(mockRouting, 0, 1)
    expect(moved.rules[0].topic).toBe('default')
    expect(moved.rules[1].topic).toBe('fast-gpu')
    expect(mockRouting.rules[0].topic).toBe('fast-gpu')
  })

  it('moveRule 边界不越界', () => {
    expect(moveRule(mockRouting, 0, -1)).toBe(mockRouting)
    expect(moveRule(mockRouting, 2, 1)).toBe(mockRouting)
  })

  it('addRule/removeRule 维护有序数组', () => {
    const added = addRule(mockRouting)
    expect(added.rules).toHaveLength(4)
    expect(added.rules[3].topic).toBeUndefined() // 新规则默认未连线
    const removed = removeRule(added, 1)
    expect(removed.rules).toHaveLength(3)
    expect(removed.rules[1].topic).toBe('batch-night')
  })
})

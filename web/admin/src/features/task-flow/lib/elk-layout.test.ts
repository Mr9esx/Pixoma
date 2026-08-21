import type { Edge, Node } from '@xyflow/react'
import { describe, expect, it } from 'vitest'
import {
  branchSourceHandle,
  branchTargetHandle,
  CASE_NODE_ID,
  CASE_SOURCE_HANDLE,
  DEFAULT_NODE_ID,
  DEFAULT_TARGET_HANDLE,
  computeFitViewport,
  elkLayout,
  topicTargetHandle,
} from './elk-layout'

function branchNode(index: number, height = 220): Node {
  return {
    id: `branch-${index}`,
    type: 'condition-branch',
    position: { x: 0, y: 0 },
    draggable: true,
    selectable: false,
    measured: { width: 560, height },
    data: {
      targetHandles: [branchTargetHandle(index)],
      sourceHandles: [branchSourceHandle(index)],
    },
  }
}

function topicNode(index: number): Node {
  return {
    id: `topic-${index}`,
    type: 'topic-target',
    position: { x: 0, y: 0 },
    selectable: false,
    measured: { width: 180, height: 38 },
    data: { targetHandles: [topicTargetHandle(`k${index}`)] },
  }
}

function baseNodes(ruleCount: number): Node[] {
  const nodes: Node[] = [
    {
      id: CASE_NODE_ID,
      type: 'case-start',
      position: { x: 0, y: 0 },
      selectable: false,
      measured: { width: 150, height: 56 },
      data: { sourceHandles: [CASE_SOURCE_HANDLE] },
    },
  ]
  for (let i = 0; i < ruleCount; i++) {
    nodes.push(branchNode(i), topicNode(i))
  }
  // 默认 Topic 放最后（与 buildGraph 约定一致：forceNodeModelOrder 下排在最右列底部）
  nodes.push({
    id: DEFAULT_NODE_ID,
    type: 'default-topic',
    position: { x: 0, y: 0 },
    selectable: false,
    measured: { width: 150, height: 38 },
    data: { targetHandles: [DEFAULT_TARGET_HANDLE] },
  })
  return nodes
}

function baseEdges(ruleCount: number): Edge[] {
  const edges: Edge[] = [
    {
      id: 'e-default',
      source: CASE_NODE_ID,
      sourceHandle: CASE_SOURCE_HANDLE,
      target: DEFAULT_NODE_ID,
      targetHandle: DEFAULT_TARGET_HANDLE,
    },
  ]
  for (let i = 0; i < ruleCount; i++) {
    edges.push(
      {
        id: `e-case-${i}`,
        source: CASE_NODE_ID,
        sourceHandle: CASE_SOURCE_HANDLE,
        target: `branch-${i}`,
        targetHandle: branchTargetHandle(i),
      },
      {
        id: `e-branch-${i}`,
        source: `branch-${i}`,
        sourceHandle: branchSourceHandle(i),
        target: `topic-${i}`,
        targetHandle: topicTargetHandle(`k${i}`),
      },
    )
  }
  return edges
}

const pos = (nodes: Node[], id: string) => nodes.find((n) => n.id === id)?.position

describe('elkLayout', () => {
  it('RIGHT 方向：Case 最左、分支居中、Topic 最右，顺序自上而下', async () => {
    const layouted = await elkLayout(baseNodes(3), baseEdges(3))
    const casePos = pos(layouted, CASE_NODE_ID)!
    const branch0 = pos(layouted, 'branch-0')!
    const branch1 = pos(layouted, 'branch-1')!
    const topic0 = pos(layouted, 'topic-0')!
    const topic1 = pos(layouted, 'topic-1')!

    expect(casePos.x).toBeLessThan(branch0.x)
    expect(branch0.x).toBeLessThan(topic0.x)
    expect(branch0.y).toBeLessThan(branch1.y)
    expect(topic0.y).toBeLessThan(topic1.y)
  })

  it('分支与对应 Topic 的纵向顺序保持一致（无交叉）', async () => {
    const layouted = await elkLayout(baseNodes(3), baseEdges(3))
    const y = (id: string) => pos(layouted, id)!.y
    expect(y('topic-0')).toBeLessThan(y('topic-1'))
    expect(y('topic-1')).toBeLessThan(y('topic-2'))
  })

  it('默认回退 Topic 固定在最右 Topic 列，排在 Topic 下方', async () => {
    const layouted = await elkLayout(baseNodes(2), baseEdges(2))
    const def = pos(layouted, DEFAULT_NODE_ID)!
    const topic0 = pos(layouted, 'topic-0')!
    const branch0 = pos(layouted, 'branch-0')!
    expect(def.x).toBe(topic0.x)
    expect(def.y).toBeGreaterThan(topic0.y)
    expect(def.x).toBeGreaterThan(branch0.x)
  })

  it('计算节点不参与图谱，按绑定 Topic 对齐到最右列', async () => {
    const nodes: Node[] = [
      ...baseNodes(1),
      {
        id: 'edge-gpu-a',
        type: 'edge-node',
        position: { x: 0, y: 0 },
        selectable: false,
        measured: { width: 160, height: 44 },
        data: { targetHandles: ['edge-gpu-a-t'] },
      },
    ]
    const edges: Edge[] = [
      ...baseEdges(1),
      { id: 'e-bind', source: 'topic-0', sourceHandle: 'topic-0-s', target: 'edge-gpu-a', targetHandle: 'edge-gpu-a-t' },
    ]
    const layouted = await elkLayout(nodes, edges)
    const edgePos = pos(layouted, 'edge-gpu-a')!
    const topicPos = pos(layouted, 'topic-0')!
    expect(edgePos.x).toBeGreaterThan(topicPos.x)
    // 垂直对齐：edge 中心 ≈ Topic 中心
    expect(edgePos.y + 22).toBeCloseTo(topicPos.y + 19, 0)
  })

  it('端口辅助函数与图边保持一致的命名契约', () => {
    expect(CASE_SOURCE_HANDLE).toBe('case-s')
    expect(branchTargetHandle(3)).toBe('branch-3-t')
    expect(branchSourceHandle(3)).toBe('branch-3-s')
    expect(topicTargetHandle('fast-gpu')).toBe('topic-fast-gpu-t')
    expect(DEFAULT_TARGET_HANDLE).toBe('default-t')
  })

  it('computeFitViewport 把图包围盒适配进视口并居中', () => {
    const nodes = [
      { id: 'a', position: { x: 0, y: 0 }, measured: { width: 100, height: 40 } },
      { id: 'b', position: { x: 100, y: 200 }, measured: { width: 100, height: 40 } },
    ] as Node[]
    const vp = computeFitViewport(nodes, 1000, 600, 0.25, 4)
    expect(vp.zoom).toBeCloseTo(Math.min((1000 * 0.5) / 200, (600 * 0.5) / 240))
    // 视口中心 = 包围盒中心
    expect(vp.x + ((0 + 200) / 2) * vp.zoom).toBeCloseTo(1000 / 2)
    expect(vp.y + ((0 + 240) / 2) * vp.zoom).toBeCloseTo(600 / 2)
  })

  it('computeFitViewport 默认 maxZoom=1，小图不会被无限放大（铺满但不溢出）', () => {
    const nodes = [{ id: 'a', position: { x: 0, y: 0 }, measured: { width: 100, height: 40 } }] as Node[]
    const vp = computeFitViewport(nodes, 1000, 600)
    expect(vp.zoom).toBeLessThanOrEqual(1)
  })
})

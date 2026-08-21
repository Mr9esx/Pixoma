import { describe, expect, it } from 'vitest'
import type { EdgePresence, EdgeRecord } from '../types'
import { isEdgeOnline, topicBindings } from './topic-binding'

const edges: EdgeRecord[] = [
  { id: 'gpu-a', name: '客厅 4090', enabled: true, subscribe_topics: ['default'], effective_topics: ['default'] },
  { id: 'gpu-b', name: '工作室双卡', enabled: true, subscribe_topics: ['fast-gpu'], effective_topics: ['fast-gpu'] },
  { id: 'gpu-c', name: '备用夜机', enabled: true, subscribe_topics: ['batch-night'], effective_topics: ['batch-night'] },
  { id: 'off', name: '停用机', enabled: false, subscribe_topics: ['fast-gpu'], effective_topics: ['fast-gpu'] },
]

const presence: EdgePresence[] = [
  { id: 'gpu-a', edge_online: true, comfy_running: true },
  { id: 'gpu-b', edge_online: true, comfy_running: true },
  { id: 'gpu-c', edge_online: false, comfy_running: false },
]

describe('topicBindings', () => {
  it('绑定 + 至少一台在线 agent = ready（可消费）', () => {
    const [def, fast] = topicBindings(edges, presence, ['default', 'fast-gpu'])
    expect(def.status).toBe('ready')
    expect(def.onlineEdges.map((e) => e.id)).toEqual(['gpu-a'])
    expect(fast.status).toBe('ready')
  })

  it('有绑定但无在线 agent = bound-offline', () => {
    const [batch] = topicBindings(edges, presence, ['batch-night'])
    expect(batch.status).toBe('bound-offline')
    expect(batch.boundEdges.map((e) => e.id)).toEqual(['gpu-c'])
    expect(batch.onlineEdges).toHaveLength(0)
  })

  it('无任何绑定 = unbound；停用 edge 不参与绑定', () => {
    const [cpuLow, fast] = topicBindings(edges, presence, ['cpu-low', 'fast-gpu'])
    expect(cpuLow.status).toBe('unbound')
    expect(fast.boundEdges.map((e) => e.id)).toEqual(['gpu-b']) // 停用的 off 被忽略
  })

  it('isEdgeOnline 要求 edge_online 与 comfy_running 同时为真', () => {
    expect(isEdgeOnline(presence, 'gpu-a')).toBe(true)
    expect(isEdgeOnline(presence, 'gpu-c')).toBe(false)
    expect(isEdgeOnline(presence, 'missing')).toBe(false)
  })
})

import type { EdgePresence, EdgeRecord } from '../types'

/**
 * Topic ↔ 计算节点（edge）绑定与可消费校验：
 * - 后台绑定 = 该 edge 的 effective_topics 包含该 Topic；
 * - 可消费（ready）= 至少一台绑定的 edge 处于在线（edge_online && comfy_running），
 *   即「后台绑定了，同时 edge-agent 也拉取了」。
 * - bound-offline = 有绑定但当前无在线 agent；unbound = 无任何绑定。
 */
export type TopicStatus = 'ready' | 'bound-offline' | 'unbound'

export interface TopicBinding {
  topic: string
  status: TopicStatus
  /** 后台已绑定（effective_topics 命中）且启用的 edge。 */
  boundEdges: EdgeRecord[]
  /** 绑定中且当前在线可拉取的 edge。 */
  onlineEdges: EdgeRecord[]
}

export function isEdgeOnline(presence: EdgePresence[], edgeId: string): boolean {
  const p = presence.find((x) => x.id === edgeId)
  return Boolean(p?.edge_online && p.comfy_running)
}

export function topicBindings(
  edges: EdgeRecord[],
  presence: EdgePresence[],
  topicKeys?: string[],
): TopicBinding[] {
  const byTopic = new Map<string, TopicBinding>()
  for (const edge of edges) {
    if (!edge.enabled) continue
    for (const topic of edge.effective_topics) {
      let binding = byTopic.get(topic)
      if (!binding) {
        binding = { topic, status: 'unbound', boundEdges: [], onlineEdges: [] }
        byTopic.set(topic, binding)
      }
      binding.boundEdges.push(edge)
      if (isEdgeOnline(presence, edge.id)) binding.onlineEdges.push(edge)
    }
  }

  const list = topicKeys
    ? topicKeys.map((key) => byTopic.get(key) ?? { topic: key, status: 'unbound' as const, boundEdges: [], onlineEdges: [] })
    : [...byTopic.values()]

  for (const binding of list) {
    binding.status =
      binding.boundEdges.length === 0
        ? 'unbound'
        : binding.onlineEdges.length > 0
          ? 'ready'
          : 'bound-offline'
  }
  return list
}

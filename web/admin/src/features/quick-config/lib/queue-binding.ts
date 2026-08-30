type EdgeLike = {
  enabled: boolean
  subscribe_topics?: string[]
  effective_topics?: string[]
}

export function queueHasSubscribers(
  edges: EdgeLike[],
  topic: string
): boolean {
  return edges.some((edge) => {
    if (!edge.enabled) return false
    const topics = [
      ...(edge.subscribe_topics ?? []),
      ...(edge.effective_topics ?? []),
    ]
    return topics.includes(topic)
  })
}

export function nodeStepCanAdvance(
  hasSubscribers: boolean,
  selectedEdgeId: string | null
): boolean {
  return hasSubscribers || selectedEdgeId != null
}

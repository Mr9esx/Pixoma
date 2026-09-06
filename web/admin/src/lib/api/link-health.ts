import { apiFetch } from './client'
import type {
  EntityHealth,
  HealthBreakpoint,
  HealthState,
  ReferenceItem,
} from '@/features/link-health/types'

export type LinkHealthKind = 'platform' | 'case' | 'topic' | 'edge'

export type LinkHealthRef = {
  id: string
  name: string
  state: HealthState
  to: string
}

export type LinkHealthNode = {
  id: string
  kind: LinkHealthKind
  ref_id: string
  name: string
  health: HealthState
  breakpoints: HealthBreakpoint[]
  upstream: LinkHealthRef[]
  downstream: LinkHealthRef[]
}

export type LinkHealthEdge = {
  from: string
  to: string
}

export type LinkHealthGraph = {
  nodes: LinkHealthNode[]
  edges: LinkHealthEdge[]
}

export function getLinkHealth() {
  return apiFetch<LinkHealthGraph>('/api/v1/link-health')
}

export function findLinkNode(
  graph: LinkHealthGraph | undefined,
  kind: LinkHealthKind,
  refId: string
): LinkHealthNode | undefined {
  return graph?.nodes.find((n) => n.kind === kind && n.ref_id === refId)
}

export function nodeEntityHealth(
  node: LinkHealthNode | undefined
): EntityHealth | undefined {
  if (!node) return undefined
  return {
    state: node.health,
    breakpoints: node.breakpoints ?? [],
  }
}

export function nodeRefs(items: LinkHealthRef[] | undefined): ReferenceItem[] {
  return (items ?? []).map((item) => ({
    id: item.id,
    name: item.name,
    state: item.state,
    to: item.to,
  }))
}

export const UNREADY_HEALTH: EntityHealth = {
  state: 'pending',
  breakpoints: [],
}

export function resolvedEntityHealth(
  graph: LinkHealthGraph | undefined,
  kind: LinkHealthKind,
  refId: string,
  ready: boolean
): EntityHealth {
  if (!ready) return UNREADY_HEALTH
  return nodeEntityHealth(findLinkNode(graph, kind, refId)) ?? UNREADY_HEALTH
}

export function healthProblems(
  health: EntityHealth | undefined,
  ready: boolean
): number {
  if (!ready || !health) return 1
  return health.state === 'ok' ? 0 : 1
}

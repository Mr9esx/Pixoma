import type { EdgePresence, RoutingConfig } from '@/features/task-flow/types'
import type { MenuPlacement } from '@/lib/api/channel-menu'

export type HealthState = 'ok' | 'warn' | 'bad'

export interface ReferenceItem {
  id: string
  name: string
  state: HealthState
  to: string
}

export interface HealthBreakpoint {
  stage: 'entry' | 'workflow' | 'topic' | 'node'
  fix: 'config' | 'runtime'
  key: string
  params?: Record<string, string>
  action: { to: string; key: string }
  guide: string
}

export interface EntityHealth {
  state: HealthState
  breakpoints: HealthBreakpoint[]
}

export interface EdgeLike {
  id: string
  name: string
  enabled: boolean
  subscribe_topics?: string[]
  effective_topics?: string[]
}

export interface CaseLike {
  id: number
  name: string
  routing?: RoutingConfig
}

export interface HealthInput {
  cases: CaseLike[]
  edges: EdgeLike[]
  presence: EdgePresence[]
  placements?: MenuPlacement[]
}

export function caseRoutingTopics(routing: RoutingConfig | undefined): string[] {
  const keys = (routing?.rules ?? [])
    .map((r) => r.topic)
    .filter((x): x is string => Boolean(x))
  return keys.length > 0 ? Array.from(new Set(keys)) : ['default']
}

export function edgeTopics(edge: EdgeLike): string[] {
  const keys =
    edge.effective_topics && edge.effective_topics.length > 0
      ? edge.effective_topics
      : (edge.subscribe_topics ?? [])
  return keys
}

export function edgeIsReady(edge: EdgeLike, presence: EdgePresence[]): boolean {
  if (!edge.enabled) return false
  const row = presence.find((p) => p.id === edge.id)
  return Boolean(row && row.edge_online && row.comfy_running)
}

const ok = (id: string, name: string, to: string): ReferenceItem => ({ id, name, state: 'ok', to })
const warn = (id: string, name: string, to: string): ReferenceItem => ({ id, name, state: 'warn', to })

export function topicReferences(key: string, input: HealthInput) {
  const cases = input.cases.filter((c) => caseRoutingTopics(c.routing).includes(key))
  const edges = input.edges.filter((e) => edgeTopics(e).includes(key))
  const ready = edges.filter((e) => edgeIsReady(e, input.presence))
  const breakpoints: HealthBreakpoint[] = []
  if (cases.length === 0) {
    breakpoints.push({
      stage: 'workflow',
      fix: 'config',
      key: 'linkHealth.noCaseRoutes',
      action: { to: '/cases', key: 'linkHealth.actionConfigureRouting' },
      guide: 'linkHealth.guideNoCaseRoutes',
    })
  }
  if (edges.length === 0) {
    breakpoints.push({
      stage: 'node',
      fix: 'config',
      key: 'linkHealth.noEdgeSubscribers',
      action: { to: '/edges', key: 'linkHealth.actionBindTopic' },
      guide: 'linkHealth.guideNoEdgeSubscribers',
    })
  } else if (ready.length === 0) {
    breakpoints.push({
      stage: 'node',
      fix: 'runtime',
      key: 'linkHealth.subscribersOffline',
      action: { to: '/edges', key: 'linkHealth.actionManageNodes' },
      guide: 'linkHealth.guideSubscribersOffline',
    })
  }
  return {
    cases: cases.map((c) => ok(String(c.id), c.name, `/cases/${c.id}`)),
    edges: edges.map((e) =>
      edgeIsReady(e, input.presence) ? ok(e.id, e.name, `/edges/${e.id}`) : warn(e.id, e.name, `/edges/${e.id}`),
    ),
    health: { state: breakpoints.length > 0 ? 'warn' : 'ok', breakpoints },
  }
}

export function edgeReferences(id: string, input: HealthInput) {
  const edge = input.edges.find((e) => e.id === id)
  if (!edge) {
    return {
      topics: [],
      cases: [],
      health: {
        state: 'bad' as HealthState,
        breakpoints: [{
          stage: 'node',
          fix: 'config',
          key: 'linkHealth.edgeMissing',
          action: { to: '/edges', key: 'linkHealth.actionManageNodes' },
          guide: 'linkHealth.guideEdgeMissing',
        }],
      },
    }
  }
  const topics = edgeTopics(edge)
  const cases = input.cases.filter((c) =>
    caseRoutingTopics(c.routing).some((k) => topics.includes(k)),
  )
  const breakpoints: HealthBreakpoint[] = []
  if (topics.length === 0) {
    breakpoints.push({
      stage: 'topic',
      fix: 'config',
      key: 'linkHealth.noTopicBinding',
      action: { to: `/edges/${id}`, key: 'linkHealth.actionEditNode' },
      guide: 'linkHealth.guideNoTopicBinding',
    })
  }
  if (cases.length === 0) {
    breakpoints.push({
      stage: 'workflow',
      fix: 'config',
      key: 'linkHealth.noCaseReachable',
      action: { to: '/cases', key: 'linkHealth.actionConfigureRouting' },
      guide: 'linkHealth.guideNoCaseReachable',
    })
  }
  if (!edgeIsReady(edge, input.presence)) {
    breakpoints.push({
      stage: 'node',
      fix: 'runtime',
      key: 'linkHealth.edgeNotReady',
      action: { to: `/edges/${id}`, key: 'linkHealth.actionCheckNode' },
      guide: 'linkHealth.guideEdgeNotReady',
    })
  }
  return {
    topics: topics.map((k) => ok(k, k, `/topics/${encodeURIComponent(k)}`)),
    cases: cases.map((c) => ok(String(c.id), c.name, `/cases/${c.id}`)),
    health: { state: breakpoints.length > 0 ? 'warn' : 'ok', breakpoints },
  }
}

export function caseReferences(caseId: number, input: HealthInput) {
  const record = input.cases.find((c) => c.id === caseId)
  if (!record) {
    return {
      menuEntries: [],
      topics: [],
      health: {
        state: 'bad' as HealthState,
        breakpoints: [{
          stage: 'workflow',
          fix: 'config',
          key: 'linkHealth.caseMissing',
          action: { to: '/cases', key: 'linkHealth.actionConfigureRouting' },
          guide: 'linkHealth.guideCaseMissing',
        }],
      },
    }
  }
  const placements = input.placements ?? []
  const topics = caseRoutingTopics(record.routing)
  const breakpoints: HealthBreakpoint[] = []
  if (placements.length === 0) {
    breakpoints.push({
      stage: 'entry',
      fix: 'config',
      key: 'linkHealth.noMenuEntry',
      action: { to: '/channels', key: 'linkHealth.actionAddEntry' },
      guide: 'linkHealth.guideNoMenuEntry',
    })
  }
  const topicItems = topics.map((key) => {
    const edges = input.edges.filter((e) => edgeTopics(e).includes(key))
    const ready = edges.filter((e) => edgeIsReady(e, input.presence))
    if (ready.length === 0) {
      breakpoints.push({
        stage: 'node',
        fix: 'runtime',
        key: 'linkHealth.topicNoReadyNode',
        params: { topic: key },
        action: { to: '/edges', key: 'linkHealth.actionManageNodes' },
        guide: 'linkHealth.guideTopicNoReadyNode',
      })
    }
    return ready.length > 0
      ? ok(key, key, `/topics/${encodeURIComponent(key)}`)
      : warn(key, key, `/topics/${encodeURIComponent(key)}`)
  })
  return {
    menuEntries: placements.map((p) =>
      ok(`${p.channel_id}:${p.item_id}`, p.channel_name ?? p.channel_id, `/channels/${encodeURIComponent(p.channel_id)}`),
    ),
    topics: topicItems,
    health: { state: breakpoints.length > 0 ? 'warn' : 'ok', breakpoints },
  }
}

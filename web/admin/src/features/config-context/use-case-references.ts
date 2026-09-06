import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  getCaseMenuPlacements,
  type MenuPlacement,
} from '@/lib/api/channel-menu'
import { listEdges, listPresence } from '@/lib/api/edges'
import {
  findLinkNode,
  nodeRefs,
  resolvedEntityHealth,
} from '@/lib/api/link-health'
import { queryKeys } from '@/lib/api/query-keys'
import {
  listRoutingAttributes,
  type RoutingAttributeDescriptor,
} from '@/lib/api/routing'
import { listTopics } from '@/lib/api/topics'
import type { CaseRecord } from '@/lib/api/types'
import { useLinkHealthQuery } from '@/features/link-health/use-link-health'
import type {
  EntityHealth,
  ReferenceItem,
} from '@/features/link-health/types'
import type {
  EdgePresence,
  EdgeRecord,
  TopicRecord,
} from '@/features/task-flow/types'

export type CaseReferencesResult = {
  menuEntries: ReferenceItem[]
  topics: ReferenceItem[]
  health: EntityHealth
}

export type CaseContextData = {
  topics: TopicRecord[]
  attributes: RoutingAttributeDescriptor[]
  edges: EdgeRecord[]
  presence: EdgePresence[]
  placements: MenuPlacement[]
  caseRefs: CaseReferencesResult | undefined
}

/** Case 详情共享数据：路由属性、节点、在线状态、菜单关联与链路健康。 */
export function useCaseReferences(record?: CaseRecord): CaseContextData {
  const topicsQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
  })
  const attributesQuery = useQuery({
    queryKey: ['routing', 'attributes'],
    queryFn: () => listRoutingAttributes(),
  })
  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })
  const placementsQuery = useQuery({
    queryKey: queryKeys.cases.menuPlacements(record?.id ?? -1),
    queryFn: () => getCaseMenuPlacements(record?.id ?? -1),
    enabled: record != null,
  })
  const healthQuery = useLinkHealthQuery()

  const topics: TopicRecord[] = topicsQuery.data ?? []
  const attributes = attributesQuery.data?.attributes ?? []
  const edges: EdgeRecord[] = (edgesQuery.data ?? []).map((e) => ({
    id: e.id,
    name: e.name,
    enabled: e.enabled,
    subscribe_topics: e.subscribe_topics ?? [],
    effective_topics: e.effective_topics ?? [],
  }))
  const presence: EdgePresence[] = presenceQuery.data ?? []
  const placements: MenuPlacement[] = placementsQuery.data ?? []

  const caseRefs = useMemo(() => {
    if (!record) return undefined
    const ready = healthQuery.isSuccess
    const node = findLinkNode(healthQuery.data, 'case', String(record.id))
    return {
      health: resolvedEntityHealth(
        healthQuery.data,
        'case',
        String(record.id),
        ready
      ),
      menuEntries: ready ? nodeRefs(node?.upstream) : [],
      topics: ready ? nodeRefs(node?.downstream) : [],
    }
  }, [record, healthQuery.data, healthQuery.isSuccess])

  return { topics, attributes, edges, presence, placements, caseRefs }
}

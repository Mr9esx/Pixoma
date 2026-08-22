import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listEdges, listPresence } from '@/lib/api/edges'
import { getCaseMenuPlacements } from '@/lib/api/channel-menu'
import { patchCase } from '@/lib/api/cases'
import { listRoutingAttributes } from '@/lib/api/routing'
import { listTopics } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import type { CaseRecord, RoutingConfig } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { TaskFlowEditor } from '@/features/task-flow/task-flow-editor'
import type { EdgePresence, EdgeRecord, TopicRecord } from '@/features/task-flow/types'
import { ContextLinks, type ContextGroup } from './context-links'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

/** Case 详情「处理流程」：编辑 routing + 关联上下文面板。 */
export function CaseContextSection({ record }: { record: CaseRecord }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [routing, setRouting] = useState<RoutingConfig | undefined>(record.routing)

  const topicsQuery = useQuery({ queryKey: queryKeys.topics.all, queryFn: () => listTopics() })
  const attributesQuery = useQuery({
    queryKey: ['routing', 'attributes'],
    queryFn: () => listRoutingAttributes(),
  })
  const edgesQuery = useQuery({ queryKey: queryKeys.edges.all, queryFn: listEdges })
  const presenceQuery = useQuery({ queryKey: queryKeys.edges.presence, queryFn: listPresence })
  const placementsQuery = useQuery({
    queryKey: queryKeys.cases.menuPlacements(record.id),
    queryFn: () => getCaseMenuPlacements(record.id),
  })

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
  const byPresence = new Map(presence.map((p) => [p.id, p]))

  const save = useMutation({
    mutationFn: () => patchCase(record.id, { routing }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.cases.detail(record.id) })
      toast.success(t('configContext.saved'))
    },
  })

  const groups: ContextGroup[] = useMemo(() => {
    const ruleTopics = Array.from(
      new Set((routing?.rules ?? []).map((r) => r.topic).filter((x): x is string => Boolean(x)))
    )
    const topicItems = ruleTopics.map((key) => {
      const topic = topics.find((x) => x.key === key)
      const online = edges.filter(
        (e) => (e.subscribe_topics ?? []).includes(key) && byPresence.get(e.id)?.edge_online
      ).length
      return {
        key,
        label: topic?.name ?? key,
        to: `/topics/${key}`,
        state: online > 0 ? ('ready' as const) : ('warn' as const),
        note: online > 0 ? t('configContext.onlineNodes', { n: online }) : t('configContext.noOnlineNode'),
      }
    })
    const nodeItems = edges
      .filter((e) => ruleTopics.some((k) => (e.subscribe_topics ?? []).includes(k)))
      .map((e) => ({
        key: e.id,
        label: e.name,
        to: `/edges/${e.id}`,
        state: (byPresence.get(e.id)?.edge_online ? 'ready' : 'warn') as ContextGroup['items'][number]['state'],
        note: byPresence.get(e.id)?.edge_online ? t('configContext.online') : t('configContext.offline'),
      }))
    const menuItems = (placementsQuery.data ?? []).map((p) => ({
      key: String(p.channel_id),
      label: p.channel_name ?? p.channel_id,
      to: `/channels/${p.channel_id}`,
      state: 'ready' as const,
      note: t('configContext.mounted'),
    }))
    return [
      { title: t('configContext.routingTopics'), items: topicItems },
      { title: t('configContext.execNodes'), items: nodeItems },
      { title: t('configContext.menuMounts'), items: menuItems },
    ].filter((g) => g.items.length > 0)
  }, [routing, topics, edges, byPresence, placementsQuery.data, t])

  return (
    <div className='space-y-4'>
      <div>
        <TaskFlowEditor
          routing={routing}
          topics={topics}
          attributes={attributes}
          edges={edges}
          presence={presence}
          caseName={record.name}
          onChange={setRouting}
          headerActions={
            <Button
              type='button'
              size='sm'
              disabled={save.isPending}
              onClick={() => save.mutate()}
              data-case-routing-save
            >
              {t('configContext.saveRouting')}
            </Button>
          }
        />
        {save.error ? (
          <p className='mt-2 text-sm text-destructive' role='alert'>
            {errorMessage(save.error)}
          </p>
        ) : null}
      </div>
      {groups.length > 0 ? <ContextLinks groups={groups} /> : null}
    </div>
  )
}

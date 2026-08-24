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
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { caseReferences } from '@/features/link-health/lib/references'
import { ConfigChain, type ChainDetail, type ChainHop } from './config-chain'

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

  const chain = useMemo(() => {
    const ruleTopics = Array.from(
      new Set((routing?.rules ?? []).map((r) => r.topic).filter((x): x is string => Boolean(x)))
    )
    const topicKeys = ruleTopics.length > 0 ? ruleTopics : ['default']
    const firstTopic = topics.find((x) => x.key === topicKeys[0])
    const topicOnline = edges.filter(
      (e) =>
        (e.subscribe_topics ?? []).includes(topicKeys[0]) &&
        byPresence.get(e.id)?.edge_online
    ).length
    const execEdges = edges
      .filter((e) => ruleTopics.some((k) => (e.subscribe_topics ?? []).includes(k)))
    const onlineEdges = execEdges.filter((e) => byPresence.get(e.id)?.edge_online)
    const placements = placementsQuery.data ?? []
    const menuHop: ChainHop = placements[0]
      ? {
          key: 'menu',
          kind: t('configChain.entry'),
          label: placements[0].channel_name ?? placements[0].channel_id,
          sub: t('configChain.mounted', { n: placements.length }),
          state: 'ok',
          to: `/channels/${placements[0].channel_id}`,
        }
      : {
          key: 'menu',
          kind: t('configChain.entry'),
          label: t('configChain.noMenu'),
          sub: '',
          state: 'warn',
          to: '/',
        }
    const topicHop: ChainHop = {
      key: 'topic',
      kind: t('configChain.delivery'),
      label: firstTopic?.name ?? topicKeys[0],
      sub:
        topicOnline > 0
          ? t('configChain.onlineNodes', { n: topicOnline })
          : t('configChain.noOnlineNode'),
      state: topicOnline > 0 ? 'ok' : 'warn',
      to: `/topics/${topicKeys[0]}`,
    }
    const nodeHop: ChainHop = {
      key: 'node',
      kind: t('configChain.exec'),
      label: onlineEdges.length > 0 ? onlineEdges.map((e) => e.name).join(' / ') : t('configChain.noOnlineNode'),
      sub:
        onlineEdges.length > 0
          ? t('configChain.onlineNodes', { n: onlineEdges.length })
          : t('configChain.noExec'),
      state: onlineEdges.length > 0 ? 'ok' : 'warn',
      to: onlineEdges[0] ? `/edges/${onlineEdges[0].id}` : '/edges',
    }
    const caseHop: ChainHop = {
      key: 'case',
      kind: t('configChain.workflow'),
      label: record.name || `#${record.id}`,
      sub: t('configChain.rules', { n: ruleTopics.length }),
      state: ruleTopics.length > 0 ? 'ok' : 'warn',
      to: `/cases/${record.id}`,
    }
    const hops = [menuHop, caseHop, topicHop, nodeHop]
    const blocked = hops.filter((h) => h.state === 'warn').length
    const health =
      blocked > 0
        ? {
            state: 'warn' as const,
            text: t('configChain.healthWarn', {
              name: record.name || `#${record.id}`,
              topic: firstTopic?.name ?? topicKeys[0],
            }),
          }
        : {
            state: 'ok' as const,
            text: t('configChain.healthOk', {
              name: record.name || `#${record.id}`,
              topic: firstTopic?.name ?? topicKeys[0],
              n: onlineEdges.length,
            }),
          }
    const topicReady = topicOnline > 0
    const details: Record<string, ChainDetail> = {
      menu: {
        conclusion:
          placements.length > 0
            ? t('configChain.menuConclusion', { n: placements.length })
            : t('configChain.noMenuConclusion'),
        rows: placements.slice(0, 3).map((p) => ({
          q: t('configChain.mountedAt'),
          a: p.channel_name ?? p.channel_id,
        })),
      },
      case: {
        conclusion: t('configChain.caseConclusion', {
          name: record.name || `#${record.id}`,
          topic: firstTopic?.name ?? topicKeys[0],
        }),
        rows: [
          { q: t('configChain.sendsTo'), a: firstTopic?.name ?? topicKeys[0] },
          { q: t('configChain.whoExecutes'), a: onlineEdges.length > 0 ? onlineEdges.map((e) => e.name).join('、') : t('configChain.noExec') },
        ],
      },
      topic: {
        conclusion: topicReady
          ? t('configChain.topicReady', { topic: firstTopic?.name ?? topicKeys[0], n: topicOnline })
          : t('configChain.topicBlocked', { topic: firstTopic?.name ?? topicKeys[0] }),
        rows: [
          { q: t('configChain.whoUses'), a: record.name || `#${record.id}` },
          { q: t('configChain.whoSubscribes'), a: topicOnline > 0 ? t('configChain.onlineNodes', { n: topicOnline }) : t('configChain.noOnlineNode') },
        ],
        actionTo: `/topics/${topicKeys[0]}`,
      },
      node: {
        conclusion:
          onlineEdges.length > 0
            ? t('configChain.nodeReady', { n: onlineEdges.length, topic: firstTopic?.name ?? topicKeys[0] })
            : t('configChain.nodeBlocked', { topic: firstTopic?.name ?? topicKeys[0] }),
        rows: execEdges.slice(0, 5).map((e) => ({
          q: t('configChain.execNodes'),
          a: `${e.name}（${byPresence.get(e.id)?.edge_online ? t('configChain.online') : t('configChain.offline')}）`,
        })),
        actionTo: onlineEdges[0] ? `/edges/${onlineEdges[0].id}` : '/edges',
      },
    }
    return { health, hops, details }
  }, [routing, topics, edges, byPresence, placementsQuery.data, record, t])

  const linkInput = {
    cases: [record],
    edges,
    presence,
    placements: placementsQuery.data ?? [],
  }
  const caseRefs = useMemo(
    () => caseReferences(record.id, linkInput),
    [record.id, linkInput],
  )

  return (
    <div className='space-y-4'>
      <LinkHealthAlert
        name={record.name}
        health={caseRefs.health}
        anchorTo='#link-health-section'
      />
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
      <ConfigChain {...chain} />
      <LinkHealthSection
        title={t('linkHealth.title')}
        health={caseRefs.health}
        upstream={{ title: t('linkHealth.menuEntries'), items: caseRefs.menuEntries }}
        downstream={{ title: t('linkHealth.topics'), items: caseRefs.topics }}
      />
    </div>
  )
}

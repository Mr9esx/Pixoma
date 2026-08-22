import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { createCase, enableCase, patchCase } from '@/lib/api/cases'
import { listEdges, listPresence } from '@/lib/api/edges'
import { listTopics } from '@/lib/api/topics'
import { getMenu, putMenu } from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { topicBindings } from '@/features/task-flow/lib/topic-binding'
import { Button } from '@/components/ui/button'
import { addWorkflowMenuEntry } from './lib/menu-payload'
import {
  computeReadiness,
  workflowStatus,
  type ReadinessLevel,
} from './lib/readiness'
import { WizardChrome } from './wizard-chrome'
import type { StepActions, WizardShared } from './types'

type Props = StepActions & { shared: WizardShared }

const READINESS_COPY: Record<ReadinessLevel, { icon: string; label: string }> = {
  ready: { icon: '✓', label: 'quickConfig.ready' },
  warn: { icon: '!', label: 'quickConfig.warnPublishable' },
  gap: { icon: '✕', label: 'quickConfig.gap' },
}

/**
 * 完成页：就绪清单 + 统一提交（Case → routing → 菜单），再发布（启用 Case）。
 * 前三步只收集草稿，这里按资源依赖顺序一次性落库。
 */
export function DoneScreen({ shared, back }: Props) {
  const { t } = useTranslation()
  const [committedId, setCommittedId] = useState<number | null>(null)
  const [published, setPublished] = useState(false)

  const topicsQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
  })
  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })

  const readiness = useMemo(() => {
    const rules = shared.routing?.rules ?? []
    const usedTopics = [
      ...new Set(
        rules
          .map((rule) => rule.topic)
          .filter((topic): topic is string => Boolean(topic))
      ),
    ]
    const mappedEdges = (edgesQuery.data ?? []).map(
      ({ id, name, enabled, subscribe_topics, effective_topics }) => ({
        id,
        name,
        enabled,
        subscribe_topics: subscribe_topics ?? [],
        effective_topics: effective_topics ?? [],
      })
    )
    const bindings = topicBindings(
      mappedEdges,
      presenceQuery.data ?? [],
      usedTopics
    )
    return computeReadiness({
      workflow: shared.caseRecord ? workflowStatus(shared.caseRecord) : 'gap',
      rules,
      enabledTopics: (topicsQuery.data ?? [])
        .filter((topic) => topic.enabled)
        .map((topic) => topic.key),
      boundTopics: bindings
        .filter((b) => b.status !== 'unbound')
        .map((b) => b.topic),
      onlineTopics: bindings
        .filter((b) => b.status === 'ready')
        .map((b) => b.topic),
      placements: [...shared.placements, ...shared.pendingEntries],
    })
  }, [
    shared.caseRecord,
    shared.routing,
    shared.placements,
    shared.pendingEntries,
    topicsQuery.data,
    edgesQuery.data,
    presenceQuery.data,
  ])

  const canPublish = Object.values(readiness).every(
    (level) => level !== 'gap'
  )

  const commitMutation = useMutation({
    mutationFn: async ({ publish }: { publish: boolean }) => {
      const draft = shared.caseRecord
      if (!draft) throw new Error('quickConfig.notReady')
      let saved =
        shared.caseId != null
          ? await patchCase(shared.caseId, draft)
          : await createCase(draft)
      if (shared.routing && shared.routing.rules.length > 0) {
        saved = await patchCase(saved.id, { ...saved, routing: shared.routing })
      }
      for (const entry of shared.pendingEntries) {
        const menu = await getMenu(entry.channelId)
        await putMenu(
          entry.channelId,
          addWorkflowMenuEntry(menu, {
            label: entry.label,
            mode: entry.mode,
            workflowId: saved.id,
          })
        )
      }
      if (publish) {
        saved = await enableCase(saved.id)
      }
      return saved
    },
    onSuccess: (saved) => {
      shared.updateCase(saved)
      setCommittedId(saved.id)
      setPublished(saved.enabled)
      toast.success(saved.enabled ? t('quickConfig.savedAndPublished') : t('quickConfig.saved'))
    },
  })

  const rows: {
    key: 'workflow' | 'processing' | 'placements'
    title: string
  }[] = [
    { key: 'workflow', title: 'quickConfig.workflowImported' },
    { key: 'processing', title: 'quickConfig.processingConfigured' },
    { key: 'placements', title: 'quickConfig.placementsTitle' },
  ]

  return (
    <WizardChrome
      step={4}
      onBack={() => back({})}
    >
      <div className='space-y-2'>
        {rows.map((row) => {
          const level = readiness[row.key]
          const copy = READINESS_COPY[level]
          return (
            <div
              key={row.key}
              className='flex items-center gap-3 rounded-md border border-border px-3 py-2.5'
              data-testid={`readiness-${row.key}`}
            >
              <span
                className={`grid size-5 shrink-0 place-items-center rounded-full text-xs font-bold ${
                  level === 'ready'
                    ? 'bg-emerald-600/15 text-emerald-600'
                    : level === 'warn'
                      ? 'bg-amber-500/15 text-amber-600'
                      : 'bg-destructive/15 text-destructive'
                }`}
              >
                {copy.icon}
              </span>
              <div className='flex-1 text-sm'>
                <span className='font-medium'>{t(row.title)}</span>
                <span className='ml-2 text-xs text-muted-foreground'>
                  {t(copy.label)}
                </span>
              </div>
            </div>
          )
        })}
      </div>

      <div className='mt-4 flex flex-wrap items-center justify-end gap-2'>
        {commitMutation.error ? (
          <span className='text-sm text-destructive'>
            {commitMutation.error instanceof Error
              ? commitMutation.error.message
              : t('quickConfig.saveFailed')}
          </span>
        ) : null}
        <Button
          type='button'
          variant='outline'
          disabled={!shared.caseRecord || commitMutation.isPending}
          onClick={() => commitMutation.mutate({ publish: false })}
        >
          {committedId != null ? t('quickConfig.resave') : t('quickConfig.saveOnly')}
        </Button>
        <Button
          type='button'
          disabled={
            !shared.caseRecord ||
            !canPublish ||
            published ||
            commitMutation.isPending
          }
          title={
            canPublish
              ? t('quickConfig.saveAllPublish')
              : t('quickConfig.gapReason')
          }
          onClick={() => commitMutation.mutate({ publish: true })}
        >
          {published ? t('quickConfig.published') : t('quickConfig.publish')}
        </Button>
        <Button type='button' variant='ghost' onClick={shared.onExit}>
          返回落地页
        </Button>
      </div>
    </WizardChrome>
  )
}

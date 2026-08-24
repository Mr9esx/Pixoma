import { useMemo, useState, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { CalendarClock, Hash, PenLine, Timer } from 'lucide-react'
import { deleteTopic, getTopic, updateTopic } from '@/lib/api/topics'
import { listCases } from '@/lib/api/cases'
import { listEdges, listPresence, patchEdge } from '@/lib/api/edges'
import type { ComfyEdge } from '@/lib/api/types'
import { queryKeys } from '@/lib/api/query-keys'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { topicReferences } from '@/features/link-health/lib/references'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { kit } from '@/features/edges/kit-classes'
import { SectionHead } from '@/features/edges/observation-panel'
import { ConfigChain, type ChainDetail, type ChainHop } from '@/features/config-context/config-chain'
import { TopicStatsPanel } from './topic-stats-panel'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function MetaChip({
  icon,
  label,
  value,
  divider = false,
}: {
  icon: ReactNode
  label: string
  value: string
  divider?: boolean
}) {
  return (
    <div className={kit.metaChip}>
      <div className='flex min-w-0 items-center gap-2 text-muted-foreground'>
        {icon}
        <span className='shrink-0'>{label}</span>
        <span className='min-w-0 truncate font-medium text-foreground'>
          {value || '—'}
        </span>
      </div>
      {divider ? (
        <div
          data-orientation='vertical'
          role='none'
          className={kit.metaChipDivider}
        />
      ) : null}
    </div>
  )
}

export function TopicDetailPanel({ topicKey }: { topicKey: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)
  const [name, setName] = useState('')

  const topicQuery = useQuery({
    queryKey: queryKeys.topics.detail(topicKey),
    queryFn: () => getTopic(topicKey),
  })
  const topic = topicQuery.data
  const isDefault = topicKey === 'default'

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.topics.detail(topicKey) })
    void queryClient.invalidateQueries({ queryKey: queryKeys.topics.all })
  }

  const updateMutation = useMutation({
    mutationFn: () => updateTopic(topicKey, { name: name.trim() || undefined }),
    onSuccess: () => {
      setEditOpen(false)
      setName('')
      invalidate()
      toast.success(t('common.successSaved'))
    },
  })

  const enableMutation = useMutation({
    mutationFn: (enabled: boolean) => updateTopic(topicKey, { enabled }),
    onSuccess: invalidate,
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteTopic(topicKey),
    onSuccess: () => {
      invalidate()
      toast.success(t('topics.deleted'))
    },
  })

  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })

  const toggleMutation = useMutation({
    mutationFn: ({ edge, bind }: { edge: ComfyEdge; bind: boolean }) => {
      const current = edge.subscribe_topics ?? []
      const next = bind
        ? [...new Set([...current, topicKey])]
        : current.filter((k) => k !== topicKey)
      return patchEdge(edge.id, { subscribe_topics: next })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
    },
  })

  const edges = edgesQuery.data ?? []
  const casesQuery = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })
  const relatedCases = (casesQuery.data ?? []).filter((c) =>
    (c.routing?.rules ?? []).some((r) => r.topic === topicKey)
  )
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })
  const linkInput = {
    cases: casesQuery.data ?? [],
    edges,
    presence: presenceQuery.data ?? [],
  }
  const topicRefs = useMemo(
    () => topicReferences(topicKey, linkInput),
    [topicKey, linkInput],
  )
  const boundEnabled = edges.filter(
    (e) => e.enabled && (e.subscribe_topics ?? []).includes(topicKey)
  )
  const topicReady = boundEnabled.length > 0
  const chainHops: ChainHop[] = [
    {
      key: 'case',
      kind: t('configChain.workflow'),
      label: relatedCases[0]?.name ?? t('configChain.noRelatedCase'),
      sub: t('configChain.rules', { n: relatedCases.length }),
      state: relatedCases.length > 0 ? 'ok' : 'warn',
      to: relatedCases[0] ? `/cases/${relatedCases[0].id}` : '/cases',
    },
    {
      key: 'topic',
      kind: t('configChain.delivery'),
      label: name || topicKey,
      sub: t('configChain.onlineNodes', { n: boundEnabled.length }),
      state: topicReady ? 'ok' : 'warn',
      to: `/topics/${topicKey}`,
    },
    {
      key: 'node',
      kind: t('configChain.exec'),
      label: boundEnabled.map((e) => e.name).join(' / ') || t('configChain.noOnlineNode'),
      sub: topicReady ? t('configChain.onlineNodes', { n: boundEnabled.length }) : t('configChain.noExec'),
      state: topicReady ? 'ok' : 'warn',
      to: boundEnabled[0] ? `/edges/${boundEnabled[0].id}` : '/edges',
    },
  ]
  const chainDetails: Record<string, ChainDetail> = {
    case: {
      conclusion: relatedCases.length > 0
        ? t('configChain.topicUsedBy', { topic: name || topicKey, n: relatedCases.length })
        : t('configChain.topicUnused', { topic: name || topicKey }),
      rows: relatedCases.slice(0, 5).map((c) => ({
        q: t('configChain.whoUses'),
        a: c.name || `#${c.id}`,
      })),
      actionTo: relatedCases[0] ? `/cases/${relatedCases[0].id}` : '/cases',
    },
    topic: {
      conclusion: topicReady
        ? t('configChain.topicReady', { topic: name || topicKey, n: boundEnabled.length })
        : t('configChain.topicBlocked', { topic: name || topicKey }),
      rows: [],
      actionTo: `/topics/${topicKey}`,
    },
    node: {
      conclusion: topicReady
        ? t('configChain.nodeReady', { n: boundEnabled.length, topic: name || topicKey })
        : t('configChain.nodeBlocked', { topic: name || topicKey }),
      rows: boundEnabled.map((e) => ({ q: t('configChain.execNodes'), a: e.name })),
      actionTo: boundEnabled[0] ? `/edges/${boundEnabled[0].id}` : '/edges',
    },
  }
  const boundFirst = [...edges].sort((a, b) => {
    const aBound = (a.subscribe_topics ?? []).includes(topicKey)
    const bBound = (b.subscribe_topics ?? []).includes(topicKey)
    return Number(bBound) - Number(aBound)
  })

  if (topicQuery.isLoading) return <LoadingSkeleton rows={8} />
  if (topicQuery.isError || !topic) {
    return (
      <ErrorBanner
        message={errorMessage(topicQuery.error) ?? t('common.errorGeneric')}
      />
    )
  }

  return (
    <section className={kit.pageSection} data-testid='topic-detail-panel'>
      <div className='flex min-w-0 flex-col gap-[6px]'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <h2 className={kit.title}>{topic.name}</h2>
            <span className={topic.enabled ? kit.tagOn : kit.tagOff}>
              {topic.enabled ? t('topics.enabled') : t('topics.disabled')}
            </span>
            {isDefault ? (
              <span className='inline-flex h-6 items-center rounded-md border border-primary/20 bg-primary/10 px-2 text-xs font-medium text-primary'>
                {t('topics.defaultBadge')}
              </span>
            ) : null}
          </div>
          <div className='flex shrink-0 flex-wrap gap-2'>
            <Button
              type='button'
              variant='outline'
              className={kit.btnGhost}
              disabled={isDefault || enableMutation.isPending}
              onClick={() => enableMutation.mutate(!topic.enabled)}
            >
              {topic.enabled ? t('topics.disable') : t('topics.enable')}
            </Button>
            <Button
              type='button'
              className={kit.btnPrimary}
              onClick={() => {
                setName(topic.name)
                setEditOpen(true)
              }}
            >
              <PenLine className='size-3.5' />
              {t('topics.editInfo')}
            </Button>
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button
                  type='button'
                  variant='destructive'
                  className='h-8 gap-1.5 rounded-md px-3 text-xs'
                  disabled={isDefault || deleteMutation.isPending}
                >
                  {t('topics.delete')}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{t('topics.deleteConfirmTitle')}</AlertDialogTitle>
                  <AlertDialogDescription>
                    {t('topics.deleteConfirmBody', { name: topic.name })}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel type='button'>
                    {t('common.cancel')}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    type='button'
                    onClick={() => deleteMutation.mutate()}
                    className='bg-destructive text-white hover:bg-destructive/90'
                  >
                    {t('topics.delete')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </div>
        <div className='mt-4 flex max-w-full flex-wrap items-center gap-2 text-xs'>
          <MetaChip
            icon={<Hash className='size-3.5' />}
            label={t('topics.fieldKey')}
            value={topic.key}
            divider
          />
          <MetaChip
            icon={<CalendarClock className='size-3.5' />}
            label={t('topics.fieldCreatedAt')}
            value={formatTime(topic.created_at)}
            divider
          />
          <MetaChip
            icon={<Timer className='size-3.5' />}
            label={t('topics.fieldUpdatedAt')}
            value={formatTime(topic.updated_at)}
          />
        </div>
      </div>

      <LinkHealthAlert
        name={name || topicKey}
        health={topicRefs.health}
        anchorTo='#link-health-section'
      />

      {updateMutation.isError ? (
        <ErrorBanner message={errorMessage(updateMutation.error)} />
      ) : null}
      {enableMutation.isError ? (
        <ErrorBanner message={errorMessage(enableMutation.error)} />
      ) : null}
      {deleteMutation.isError ? (
        <ErrorBanner message={errorMessage(deleteMutation.error)} />
      ) : null}

      {isDefault ? (
        <p className='rounded-md border border-border bg-muted/30 p-3 text-xs text-muted-foreground'>
          {t('topics.defaultHint')}
        </p>
      ) : null}

      <section>
        <SectionHead
          title={t('topics.boundNodes')}
          hint={t('topics.boundNodesHint')}
        />
        <div className='mt-4 overflow-hidden rounded-[8px] border bg-card shadow-sm shadow-zinc-200/40 dark:shadow-none'>
          {edgesQuery.isError ? (
            <div className='p-4'>
              <ErrorBanner message={errorMessage(edgesQuery.error)} />
            </div>
          ) : null}
          {edgesQuery.isLoading ? (
            <div className='p-4'>
              <LoadingSkeleton rows={3} />
            </div>
          ) : null}
          {!edgesQuery.isLoading && !edgesQuery.isError && edges.length === 0 ? (
            <p className='p-4 text-sm text-muted-foreground'>
              {t('topics.noNodes')}
            </p>
          ) : null}
          {!edgesQuery.isLoading && !edgesQuery.isError && edges.length > 0 ? (
            <ul className='divide-y'>
              {boundFirst.map((edge) => {
                const bound = (edge.subscribe_topics ?? []).includes(topicKey)
                const pending =
                  toggleMutation.isPending &&
                  toggleMutation.variables?.edge.id === edge.id
                return (
                  <li
                    key={edge.id}
                    className='flex items-center justify-between gap-2 px-4 py-2.5'
                  >
                    <div className='flex min-w-0 items-center gap-2'>
                      <span
                        className={
                          bound ? kit.healthDot.ok : 'size-2 shrink-0 rounded-full bg-muted-foreground/40'
                        }
                      />
                      <span className='truncate text-sm'>{edge.name}</span>
                    </div>
                    <Button
                      type='button'
                      variant={bound ? 'outline' : 'default'}
                      className={bound ? kit.btnGhost : 'h-8 gap-1.5 rounded-md px-3 text-xs'}
                      disabled={pending}
                      onClick={() => toggleMutation.mutate({ edge, bind: !bound })}
                    >
                      {bound ? t('topics.unbind') : t('topics.bind')}
                    </Button>
                  </li>
                )
              })}
            </ul>
          ) : null}
        </div>
      </section>

      <section>
        <SectionHead
          title={t('topics.statsTitle')}
          hint={t('topics.statsHint')}
        />
        <div className='mt-4'>
          <TopicStatsPanel topicKey={topicKey} />
        </div>
      </section>

      <section className='mt-4'>
        <ConfigChain
          health={
            topicReady
              ? { state: 'ok', text: t('configChain.healthTopicOk', { topic: name || topicKey, n: boundEnabled.length }) }
              : { state: 'warn', text: t('configChain.healthTopicWarn', { topic: name || topicKey }) }
          }
          hops={chainHops}
          details={chainDetails}
        />
      </section>

      <LinkHealthSection
        title={t('linkHealth.title')}
        health={topicRefs.health}
        upstream={{ title: t('linkHealth.referencingCases'), items: topicRefs.cases }}
        downstream={{ title: t('linkHealth.edges'), items: topicRefs.edges }}
      />

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='sm:max-w-md'>
          <DialogHeader>
            <DialogTitle>{t('topics.editInfo')}</DialogTitle>
          </DialogHeader>
          <div className='space-y-1.5'>
            <Label htmlFor='topic-name'>{t('topics.fieldName')}</Label>
            <Input
              id='topic-name'
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoComplete='off'
            />
          </div>
          <div className='flex justify-end gap-2'>
            <Button
              type='button'
              variant='outline'
              onClick={() => setEditOpen(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type='button'
              disabled={updateMutation.isPending || !name.trim()}
              onClick={() => updateMutation.mutate()}
            >
              {t('common.save')}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </section>
  )
}

function formatTime(iso: string): string {
  const ms = Date.parse(iso)
  if (!Number.isFinite(ms)) return iso
  return new Date(ms).toLocaleString()
}

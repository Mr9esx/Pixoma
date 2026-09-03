import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  CalendarCheck,
  Hash,
  MoreHorizontal,
  PenLine,
  Power,
  SearchX,
  Timer,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listCases } from '@/lib/api/cases'
import { ApiError } from '@/lib/api/client'
import { listEdges, listPresence } from '@/lib/api/edges'
import { topicDeleteErrorMessage } from '@/lib/api/localized-errors'
import { queryKeys } from '@/lib/api/query-keys'
import { listTasks } from '@/lib/api/tasks'
import { deleteTopic, getTopic, updateTopic } from '@/lib/api/topics'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Reveal } from '@/components/ui/reveal'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { NotFoundState } from '@/components/feedback/not-found-state'
import { MetaChip } from '@/components/meta-chip'
import { SectionHead } from '@/components/section-head'
import { kit } from '@/features/edges/kit-classes'
import { topicReferences } from '@/features/link-health/lib/references'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { TopologyOpenButton } from '@/features/config-topology/topology-dialog'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { TopicStatsPanel } from './topic-stats-panel'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function TopicDetailPanel({ topicKey }: { topicKey: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [name, setName] = useState('')

  const topicQuery = useQuery({
    queryKey: queryKeys.topics.detail(topicKey),
    queryFn: () => getTopic(topicKey),
  })
  const topic = topicQuery.data
  const isDefault = topicKey === 'default'

  const invalidate = () => {
    void queryClient.invalidateQueries({
      queryKey: queryKeys.topics.detail(topicKey),
    })
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
    meta: { handledError: true },
    mutationFn: () => deleteTopic(topicKey, ackImpact),
    onSuccess: async (summary) => {
      invalidate()
      toast.success(
        summary?.removed_case_rules !== undefined
          ? t('topics.deleteDone', {
              rules: summary.removed_case_rules ?? 0,
              nodes: summary.removed_edge_subscriptions ?? 0,
              tasks: summary.failed_tasks ?? 0,
            })
          : t('topics.deleted')
      )
    },
    onError: (err) => {
      toast.error(
        topicDeleteErrorMessage(err, t) ??
          errorMessage(err) ??
          t('topics.deletedFailed')
      )
    },
  })

  const [ackImpact, setAckImpact] = useState(false)
  const queuedTasksQuery = useQuery({
    queryKey: ['topics', topicKey, 'queued-tasks'] as const,
    queryFn: () => listTasks({ dispatch_topic: topicKey, status: 'queued' }),
  })
  const queuedCount = queuedTasksQuery.data?.length ?? 0

  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })

  const edges = edgesQuery.data ?? []
  const casesQuery = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })
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
    [topicKey, linkInput]
  )
  const caseRefCount = topicRefs.cases.length
  const edgeRefCount = topicRefs.edges.length
  const hasImpact = caseRefCount > 0 || edgeRefCount > 0 || queuedCount > 0

  if (topicQuery.isLoading) {
    return (
      <div className={kit.pageSection}>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }
  if (topicQuery.isError || !topic) {
    const notFound =
      (topicQuery.error instanceof ApiError &&
        topicQuery.error.status === 404) ||
      (!topic && !topicQuery.error)
    if (notFound) {
      return (
        <NotFoundState
          icon={<SearchX />}
          title={t('topics.notFoundTitle')}
          description={t('topics.notFoundDesc')}
          actions={
            <>
              <Button asChild size='sm'>
                <Link to='/topics'>{t('topics.backToList')}</Link>
              </Button>
              <Button asChild variant='outline' size='sm'>
                <Link to='/topics/$key' params={{ key: 'new' }}>
                  {t('topics.new')}
                </Link>
              </Button>
            </>
          }
        />
      )
    }
    return (
      <div className={kit.pageSection}>
        <ErrorBanner
          message={errorMessage(topicQuery.error) ?? t('common.errorGeneric')}
          onRetry={() => void topicQuery.refetch()}
        />
      </div>
    )
  }

  return (
    <Reveal
      as='section'
      className={kit.pageSection}
      data-testid='topic-detail-panel'
    >
      <div className='flex min-w-0 flex-col gap-[6px]'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <h2 className={kit.title}>{topic.name}</h2>
            {isDefault ? (
              <Badge className='border-primary/20 bg-primary/10 text-primary'>
                {t('topics.defaultBadge')}
              </Badge>
            ) : null}
          </div>
          <div className='flex shrink-0 flex-wrap gap-2'>
            <TopologyOpenButton kind='topic' id={topic.key} />
            <Button
              type='button'
              size='sm'
              onClick={() => {
                setName(topic.name)
                setEditOpen(true)
              }}
            >
              <PenLine className='size-3.5' strokeWidth={2} />
              {t('topics.edit')}
            </Button>
            <Button
              type='button'
              variant='outline'
              size='sm'
              disabled={isDefault || enableMutation.isPending}
              onClick={() => enableMutation.mutate(!topic.enabled)}
            >
              <Power className='size-3.5' strokeWidth={2} />
              {topic.enabled ? t('topics.disable') : t('topics.enable')}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type='button'
                  variant='outline'
                  size='icon-sm'
                  aria-label={t('common.moreActions')}
                >
                  <MoreHorizontal className='size-3.5' strokeWidth={2} />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align='end'>
                <DropdownMenuItem
                  variant='destructive'
                  disabled={isDefault || deleteMutation.isPending}
                  onSelect={() => setDeleteOpen(true)}
                >
                  <Trash2 className='size-3.5' strokeWidth={2} />
                  {t('topics.delete')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            <ConfirmDialog
              open={deleteOpen}
              onOpenChange={setDeleteOpen}
              destructive
              isLoading={deleteMutation.isPending}
              disabled={hasImpact ? !ackImpact : false}
              title={t('topics.deleteConfirmTitle')}
              desc={
                <div className='text-sm'>
                  {t('topics.deleteConfirmBody', { name: topic.name })}
                  {caseRefCount > 0 ? (
                    <p className='mt-3'>
                      {t('topics.deleteWillRemoveRules', {
                        count: caseRefCount,
                      })}
                    </p>
                  ) : null}
                  {edgeRefCount > 0 ? (
                    <p className='mt-1'>
                      {t('topics.deleteWillUnbindNodes', {
                        count: edgeRefCount,
                      })}
                    </p>
                  ) : null}
                  {queuedCount > 0 ? (
                    <p className='mt-1'>
                      {t('topics.deleteWillFailQueued', { count: queuedCount })}
                    </p>
                  ) : null}
                </div>
              }
              confirmText={t('topics.delete')}
              cancelBtnText={t('common.cancel')}
              handleConfirm={() => deleteMutation.mutate()}
            >
              {hasImpact ? (
                <label className='flex cursor-pointer items-start gap-2 text-sm'>
                  <Checkbox
                    checked={ackImpact}
                    onCheckedChange={(checked) =>
                      setAckImpact(checked === true)
                    }
                    data-testid='topic-delete-ack'
                  />
                  <span>{t('topics.deleteAckImpact')}</span>
                </label>
              ) : null}
            </ConfirmDialog>
          </div>
        </div>
        <div className='mt-1 flex max-w-full flex-wrap items-center gap-2 text-xs'>
          <MetaChip
            icon={<Hash className='size-3.5' strokeWidth={2} />}
            label={t('topics.fieldKey')}
            value={topic.key}
            divider
          />
          <MetaChip
            icon={<CalendarCheck className='size-3.5' strokeWidth={2} />}
            label={t('topics.fieldCreatedAt')}
            value={formatTime(topic.created_at)}
            divider
          />
          <MetaChip
            icon={<Timer className='size-3.5' strokeWidth={2} />}
            label={t('topics.fieldUpdatedAt')}
            value={formatTime(topic.updated_at)}
          />
        </div>
      </div>

      <div className='flex flex-col gap-2'>
        <LinkHealthAlert
          name={name || topicKey}
          health={topicRefs.health}
          anchorTo='#link-health-section'
        />

        {isDefault ? (
          <Alert variant='default'>
            <AlertDescription>{t('topics.defaultHint')}</AlertDescription>
          </Alert>
        ) : null}
      </div>

      <section className='flex flex-col gap-4'>
        <SectionHead
          title={t('topics.statsTitle')}
          hint={t('topics.statsHint')}
        />
        <TopicStatsPanel topicKey={topicKey} />
      </section>

      <LinkHealthSection
        title={t('linkHealth.title')}
        health={topicRefs.health}
        upstream={{
          title: t('linkHealth.usedWorkflows'),
          items: topicRefs.cases,
        }}
        downstream={{
          title: t('linkHealth.boundNodes'),
          items: topicRefs.edges,
        }}
      />

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='sm:max-w-md'>
          <DialogHeader>
            <DialogTitle>{t('topics.editInfo')}</DialogTitle>
          </DialogHeader>
          <FieldGroup className='gap-4'>
            <Field>
              <FieldLabel htmlFor='topic-name'>
                {t('topics.fieldName')}
              </FieldLabel>
              <Input
                id='topic-name'
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoComplete='off'
              />
            </Field>
          </FieldGroup>
          <DialogFooter>
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
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Reveal>
  )
}

function formatTime(iso: string): string {
  const ms = Date.parse(iso)
  if (!Number.isFinite(ms)) return iso
  return new Date(ms).toLocaleString()
}

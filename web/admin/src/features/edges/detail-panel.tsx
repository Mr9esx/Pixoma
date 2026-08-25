import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  BadgeCheck,
  Boxes,
  CalendarClock,
  Cpu,
  Gpu,
  Hash,
  ListTodo,
  MemoryStick,
  PenLine,
  Tags,
  Terminal,
  Timer,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { listCases } from '@/lib/api/cases'
import {
  deleteEdge,
  getEdge,
  getEdgeMetrics,
  getEdgeStats,
  listEdgeTasks,
  listPresence,
  type MetricsRange,
} from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import type { ComfyEdge } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { LongText } from '@/components/long-text'
import { MetaChip } from '@/components/meta-chip'
import { SectionHead } from '@/components/section-head'
import { edgeReferences } from '@/features/link-health/lib/references'
import { LinkHealthAlert } from '@/features/link-health/link-health-alert'
import { LinkHealthSection } from '@/features/link-health/link-health-section'
import { AgentCredentials } from './agent-credentials'
import { EdgeForm } from './edge-form'
import { kit } from './kit-classes'
import { formatBytes } from './observation'
import { ObservationPanel } from './observation-panel'
import { StatusTag } from './presence-tags'

const TASKS_PAGE_SIZE = 10

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function formatTime(iso: string): string {
  const ms = Date.parse(iso)
  if (!Number.isFinite(ms)) return iso
  return new Date(ms).toLocaleString()
}

function formatRuntime(ms: number): string {
  if (ms < 1000) return `${ms} ms`
  const seconds = ms / 1000
  if (seconds < 60) return `${seconds.toFixed(1)} s`
  const minutes = seconds / 60
  if (minutes < 60) return `${minutes.toFixed(1)} m`
  return `${(minutes / 60).toFixed(1)} h`
}

function formatRate(rate: number | null): string {
  if (rate == null || !Number.isFinite(rate)) return '—'
  return `${(rate * 100).toFixed(1)}%`
}

type Props = {
  id: string
}

export function EdgeDetailPanel({ id }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)
  const [deployOpen, setDeployOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [ackRunning, setAckRunning] = useState(false)
  const [tasksOffset, setTasksOffset] = useState(0)
  const [metricsRange, setMetricsRange] = useState<MetricsRange>({
    kind: 'preset',
    window: '1h',
  })

  const detailQuery = useQuery({
    queryKey: queryKeys.edges.detail(id),
    queryFn: () => getEdge(id),
  })
  const statsQuery = useQuery({
    queryKey: queryKeys.edges.stats(id),
    queryFn: () => getEdgeStats(id),
  })
  const tasksQuery = useQuery({
    queryKey: queryKeys.edges.tasks(id, tasksOffset),
    queryFn: () =>
      listEdgeTasks(id, { limit: TASKS_PAGE_SIZE, offset: tasksOffset }),
  })
  const metricsQuery = useQuery({
    queryKey: queryKeys.edges.metrics(
      id,
      metricsRange.kind === 'preset'
        ? metricsRange.window
        : `custom:${metricsRange.from}:${metricsRange.to}`
    ),
    queryFn: () =>
      metricsRange.kind === 'preset'
        ? getEdgeMetrics(id, metricsRange.window)
        : getEdgeMetrics(id, 'custom', {
            from: metricsRange.from,
            to: metricsRange.to,
          }),
    refetchInterval: 15000,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
    refetchInterval: 5000,
  })
  const casesQuery = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases(),
  })

  const runningTasksQuery = useQuery({
    queryKey: ['edges', id, 'running-tasks'] as const,
    queryFn: () => listEdgeTasks(id, { status: 'running' }),
    enabled: deleteOpen,
  })
  const runningCount = runningTasksQuery.data?.length ?? 0

  const deleteMutation = useMutation({
    mutationFn: () => deleteEdge(id, ackRunning),
    onSuccess: async (summary) => {
      toast.success(
        summary.failed_tasks
          ? t('edges.deleteDone', { count: summary.failed_tasks })
          : t('edges.deleteSuccess')
      )
      await queryClient.invalidateQueries({ queryKey: queryKeys.edges.all })
      queryClient.removeQueries({ queryKey: queryKeys.edges.detail(id) })
      void navigate({ to: '/edges' })
    },
    onError: (err) => {
      toast.error(errorMessage(err) ?? t('edges.deleteFailed'))
    },
  })
  const edgeRefs = useMemo(() => {
    const data = detailQuery.data
    return edgeReferences(id, {
      cases: casesQuery.data ?? [],
      edges: data
        ? [
            {
              id: data.id,
              name: data.name,
              enabled: data.enabled,
              subscribe_topics: data.subscribe_topics ?? [],
              effective_topics: data.effective_topics ?? [],
            },
          ]
        : [],
      presence: presenceQuery.data ?? [],
    })
  }, [id, detailQuery.data, casesQuery.data, presenceQuery.data])

  if (detailQuery.isError) {
    return (
      <div className={kit.pageSection}>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }
  if (detailQuery.isLoading || !detailQuery.data) {
    return (
      <div className={kit.pageSection}>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  const edge = detailQuery.data
  const presence = presenceQuery.data?.find((row) => row.id === edge.id)
  const hardware = edge.hardware
  const gpus = hardware?.gpus?.filter((gpu) => gpu.name.trim()) ?? []
  const cpuModel = hardware?.cpu_model ?? ''
  const cpuCores = hardware?.cpu_cores ?? undefined
  const ramValue =
    hardware?.ram_bytes && hardware.ram_bytes > 0
      ? formatBytes(hardware.ram_bytes)
      : ''
  const stats = statsQuery.data

  return (
    <section className={kit.pageSection} data-testid='edge-detail'>
      <div className='flex min-w-0 flex-col gap-[6px]'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <h2 className={kit.title}>{edge.name || edge.id}</h2>
            <StatusTag on={edge.enabled}>
              {edge.enabled ? t('edges.enabled') : t('edges.disabled')}
            </StatusTag>
            <StatusTag on={presence?.edge_online === true}>
              {presence?.edge_online
                ? t('edges.nodeOnline')
                : t('edges.nodeOffline')}
            </StatusTag>
            <StatusTag on={presence?.comfy_running === true}>
              {presence?.comfy_running
                ? t('edges.comfyRunning')
                : t('edges.comfyStopped')}
            </StatusTag>
          </div>
          <div className='flex shrink-0 flex-wrap gap-2'>
            <Button
              type='button'
              variant='outline'
              className={kit.btnGhost}
              onClick={() => setDeployOpen(true)}
            >
              <Terminal className='size-3.5' />
              {t('edges.deployCommand')}
            </Button>
            <Button
              type='button'
              className={kit.btnPrimary}
              onClick={() => setEditOpen(true)}
            >
              <PenLine className='size-3.5' />
              {t('edges.edit')}
            </Button>
            <Button
              type='button'
              variant='outline'
              className='h-8 gap-1.5 px-3 text-xs text-destructive hover:text-destructive'
              onClick={() => setDeleteOpen(true)}
            >
              <Trash2 className='size-3.5' />
              {t('common.delete')}
            </Button>
          </div>
        </div>
        {edge.description ? (
          <LongText className='max-w-full text-sm text-muted-foreground'>
            {edge.description}
          </LongText>
        ) : null}
        <div className='mt-4 flex max-w-full flex-wrap items-center gap-2 text-xs'>
          <MetaChip
            icon={<Hash className='size-3.5' />}
            label={t('edges.fieldNodeId')}
            value={edge.id}
            divider
          />
          <MetaChip
            icon={<CalendarClock className='size-3.5' />}
            label={t('edges.fieldCreatedAt')}
            value={formatTime(edge.created_at)}
            divider
          />
          <MetaChip
            icon={<Timer className='size-3.5' />}
            label={t('edges.fieldStartedAt')}
            value={
              edge.enabled && presence?.edge_online && edge.started_at
                ? formatTime(edge.started_at)
                : '—'
            }
            divider
          />
          <MetaChip
            icon={<Boxes className='size-3.5' />}
            label={t('edges.fieldComfyVersion')}
            value={edge.comfy_version}
            divider
          />
          <MetaChip
            icon={<Tags className='size-3.5' />}
            label={t('edges.fieldCapabilities')}
            value={edge.capabilities.join(', ')}
          />
        </div>
      </div>

      <LinkHealthAlert
        name={edge.name}
        health={edgeRefs.health}
        anchorTo='#link-health-section'
      />

      <section className={kit.specsWrap}>
        <div className={kit.specsCell}>
          <div className={kit.specsLabel}>
            <Cpu className='size-3.5 shrink-0' />
            {t('edges.fieldCpu')}
          </div>
          <div className='mt-2 flex min-w-0 flex-col gap-1 xl:flex-row xl:items-end xl:justify-between xl:gap-3'>
            <LongText className={kit.specsValue}>{cpuModel || '—'}</LongText>
          </div>
        </div>
        <div className={kit.specsCell}>
          <div className={kit.specsLabel}>
            <Gpu className='size-3.5 shrink-0' />
            {t('edges.fieldGpu')}
          </div>
          {gpus.length > 0 ? (
            <div className='mt-2 flex min-w-0 flex-col gap-2'>
              {gpus.map((gpu, index) => (
                <div
                  key={`${gpu.name}-${index}`}
                  className='flex min-w-0 flex-col gap-1 xl:flex-row xl:items-end xl:justify-between xl:gap-3'
                >
                  <LongText className={kit.specsValue}>{gpu.name}</LongText>
                  <p className={kit.specsNote}>
                    {gpu.vram_bytes && gpu.vram_bytes > 0
                      ? formatBytes(gpu.vram_bytes)
                      : '—'}
                  </p>
                </div>
              ))}
            </div>
          ) : (
            <div className='mt-2 flex min-w-0 flex-col gap-1 xl:flex-row xl:items-end xl:justify-between xl:gap-3'>
              <LongText className={kit.specsValue}>—</LongText>
            </div>
          )}
        </div>
        <div className={kit.specsCell}>
          <div className={kit.specsLabel}>
            <Cpu className='size-3.5 shrink-0' />
            {t('edges.fieldCpuCores')}
          </div>
          <div className='mt-2 flex min-w-0 flex-col gap-1 xl:flex-row xl:items-end xl:justify-between xl:gap-3'>
            <LongText className={kit.specsValue}>
              {cpuCores && cpuCores > 0
                ? t('edges.coresValue', { count: cpuCores })
                : '—'}
            </LongText>
          </div>
        </div>
        <div className={kit.specsCell}>
          <div className={kit.specsLabel}>
            <MemoryStick className='size-3.5 shrink-0' />
            {t('edges.fieldMemory')}
          </div>
          <div className='mt-2 flex min-w-0 flex-col gap-1 xl:flex-row xl:items-end xl:justify-between xl:gap-3'>
            <LongText className={kit.specsValue}>{ramValue || '—'}</LongText>
          </div>
        </div>
      </section>

      <section className='flex flex-col gap-4'>
        <SectionHead
          title={t('edges.overviewTitle')}
          hint={t('edges.overviewHint')}
        />
        <div className={kit.statsWrap}>
          <div className={kit.statsGrid}>
            <div className={kit.statsCell[0]}>
              <p className={kit.statsLabel}>
                <ListTodo className='size-4' />
                {t('edges.statsTasks')}
              </p>
              <p className={kit.statsValue}>
                {statsQuery.isLoading ? '—' : String(stats?.task_count ?? 0)}
              </p>
            </div>
            <div className={kit.statsCell[1]}>
              <p className={kit.statsLabel}>
                <Timer className='size-4' />
                {t('edges.statsRuntime')}
              </p>
              <p className={kit.statsValue}>
                {statsQuery.isLoading
                  ? '—'
                  : formatRuntime(stats?.runtime_ms ?? 0)}
              </p>
            </div>
            <div className={kit.statsCell[2]}>
              <p className={kit.statsLabel}>
                <BadgeCheck className='size-4' />
                {t('edges.statsSuccess')}
              </p>
              <p className={kit.statsValue}>
                {statsQuery.isLoading
                  ? '—'
                  : formatRate(stats?.success_rate ?? null)}
              </p>
            </div>
          </div>
        </div>
      </section>

      <ObservationPanel
        metricsQuery={metricsQuery}
        metricsRange={metricsRange}
        onMetricsRangeChange={setMetricsRange}
        tasksQuery={tasksQuery}
        tasksPagination={{
          offset: tasksOffset,
          pageSize: TASKS_PAGE_SIZE,
          onOffsetChange: setTasksOffset,
        }}
      />

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-lg'>
          <DialogHeader className='shrink-0'>
            <DialogTitle>{t('edges.editHeading')}</DialogTitle>
          </DialogHeader>
          <EdgeForm
            mode='edit'
            initial={edge}
            onSaved={(next: ComfyEdge) => {
              queryClient.setQueryData(queryKeys.edges.detail(id), next)
              setEditOpen(false)
            }}
            onDeleted={() => {
              setEditOpen(false)
              void navigate({ to: '/edges' })
            }}
          />
        </DialogContent>
      </Dialog>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('edges.deleteConfirmTitle')}</DialogTitle>
          </DialogHeader>
          <div className='space-y-3 text-sm'>
            {runningCount > 0 ? (
              <p>{t('edges.deleteWillFailRunning', { count: runningCount })}</p>
            ) : (
              <p>{t('edges.deleteNoRunning')}</p>
            )}
            {(edge.subscribe_topics ?? []).length > 0 ? (
              <div className='space-y-1'>
                <p className='font-medium'>
                  {t('edges.deleteSubscribedTopics')}
                </p>
                <ul className='max-h-32 overflow-auto rounded-md border bg-muted/20 p-3 text-xs'>
                  {(edge.subscribe_topics ?? []).map((tp) => (
                    <li key={tp}>{tp}</li>
                  ))}
                </ul>
              </div>
            ) : null}
            {runningCount > 0 ? (
              <label className='flex items-start gap-2'>
                <input
                  type='checkbox'
                  checked={ackRunning}
                  onChange={(e) => setAckRunning(e.target.checked)}
                  data-testid='edge-delete-ack'
                />
                <span>{t('edges.deleteAckRunning')}</span>
              </label>
            ) : null}
          </div>
          <DialogFooter>
            <Button
              type='button'
              variant='outline'
              onClick={() => setDeleteOpen(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              type='button'
              variant='destructive'
              disabled={
                deleteMutation.isPending || (runningCount > 0 && !ackRunning)
              }
              onClick={() => deleteMutation.mutate()}
            >
              {t('common.delete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deployOpen} onOpenChange={setDeployOpen}>
        <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('edges.deployCommand')}</DialogTitle>
          </DialogHeader>
          <AgentCredentials
            edge={edge}
            onEdgeChange={(next) => {
              queryClient.setQueryData(queryKeys.edges.detail(id), next)
            }}
          />
        </DialogContent>
      </Dialog>

      <LinkHealthSection
        title={t('linkHealth.title')}
        health={edgeRefs.health}
        upstream={{
          title: t('linkHealth.executedWorkflows'),
          items: edgeRefs.cases,
        }}
        downstream={{
          title: t('linkHealth.subscribedTopics'),
          items: edgeRefs.topics,
        }}
      />
    </section>
  )
}

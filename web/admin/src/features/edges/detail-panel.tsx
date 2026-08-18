import { useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
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
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
  getEdge,
  getEdgeMetrics,
  getEdgeStats,
  listEdgeTasks,
  listPresence,
} from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import type { ComfyEdge } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { LongText } from '@/components/long-text'
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

function InfoField({
  icon,
  label,
  value,
}: {
  icon: ReactNode
  label: string
  value: string
}) {
  return (
    <div className='grid grid-cols-[7rem_minmax(0,1fr)] items-start gap-1'>
      <dt className={kit.dt}>
        {icon}
        {label}
      </dt>
      <dd className={kit.dd}>
        <LongText className='min-w-0 font-medium'>{value || '—'}</LongText>
      </dd>
    </div>
  )
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
  const [tasksOffset, setTasksOffset] = useState(0)

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
    queryKey: queryKeys.edges.metrics(id),
    queryFn: () => getEdgeMetrics(id, '1h'),
    refetchInterval: 15000,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
    refetchInterval: 5000,
  })

  if (detailQuery.isError) {
    return (
      <div className='p-6'>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }
  if (detailQuery.isLoading || !detailQuery.data) {
    return (
      <div className='p-6'>
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
      <div className={kit.header}>
        <div className='flex min-w-0 flex-col gap-[6px]'>
          <div className='flex flex-wrap items-center gap-2'>
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
          {edge.description ? (
            <p className={kit.desc}>{edge.description}</p>
          ) : null}
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
        </div>
      </div>

      <div data-orientation='horizontal' role='none' className={kit.rule} />

      <dl className={kit.dl}>
        <div className='flex flex-col gap-4'>
          <InfoField
            icon={<Hash className='size-4' />}
            label={t('edges.fieldId')}
            value={edge.id}
          />
          <InfoField
            icon={<CalendarClock className='size-4' />}
            label={t('edges.fieldCreatedAt')}
            value={formatTime(edge.created_at)}
          />
          <InfoField
            icon={<Timer className='size-4' />}
            label={t('edges.fieldStartedAt')}
            value={edge.started_at ? formatTime(edge.started_at) : '—'}
          />
          <InfoField
            icon={<Boxes className='size-4' />}
            label={t('edges.fieldComfyVersion')}
            value={edge.comfy_version || '—'}
          />
          <InfoField
            icon={<Tags className='size-4' />}
            label={t('edges.fieldCapabilities')}
            value={edge.capabilities.join(', ')}
          />
        </div>
        <div className='flex flex-col gap-4'>
          <InfoField
            icon={<Cpu className='size-4' />}
            label={t('edges.fieldCpu')}
            value={cpuModel}
          />
          {cpuCores && cpuCores > 0 ? (
            <InfoField
              icon={<Cpu className='size-4' />}
              label={t('edges.fieldCpuCores')}
              value={t('edges.coresValue', { count: cpuCores })}
            />
          ) : null}
          <InfoField
            icon={<MemoryStick className='size-4' />}
            label={t('edges.fieldMemory')}
            value={ramValue}
          />
          {gpus.length > 0 ? (
            gpus.flatMap((gpu, index) => {
              const suffix = gpus.length > 1 ? ` ${index + 1}` : ''
              return [
                <InfoField
                  key={`gpu-${index}`}
                  icon={<Gpu className='size-4' />}
                  label={`${t('edges.fieldGpu')}${suffix}`}
                  value={gpu.name}
                />,
                <InfoField
                  key={`vram-${index}`}
                  icon={<Gpu className='size-4' />}
                  label={`${t('edges.fieldVram')}${suffix}`}
                  value={
                    gpu.vram_bytes && gpu.vram_bytes > 0
                      ? formatBytes(gpu.vram_bytes)
                      : '—'
                  }
                />,
              ]
            })
          ) : (
            <InfoField
              icon={<Gpu className='size-4' />}
              label={t('edges.fieldGpu')}
              value=''
            />
          )}
        </div>
      </dl>

      <div data-orientation='horizontal' role='none' className={kit.rule} />

      <div className={kit.statsWrap}>
        <div className={kit.statsGrid}>
          <div className={kit.statsCell}>
            <p className={kit.statsLabel}>
              <ListTodo className='size-4' />
              {t('edges.statsTasks')}
            </p>
            <p className={kit.statsValue}>
              {statsQuery.isLoading ? '—' : String(stats?.task_count ?? 0)}
            </p>
          </div>
          <div className={kit.statsCell}>
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
          <div className={kit.statsCell}>
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

      <ObservationPanel
        metricsQuery={metricsQuery}
        tasksQuery={tasksQuery}
        tasksPagination={{
          offset: tasksOffset,
          pageSize: TASKS_PAGE_SIZE,
          onOffsetChange: setTasksOffset,
        }}
      />

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-lg'>
          <DialogHeader>
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
    </section>
  )
}

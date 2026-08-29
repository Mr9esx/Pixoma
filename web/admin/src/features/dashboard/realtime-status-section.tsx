import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from 'recharts'
import { listEdges, listPresence } from '@/lib/api/edges'
import { listFleetStats } from '@/lib/api/stats'
import { queryKeys } from '@/lib/api/query-keys'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from '@/components/ui/chart'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { CountUp } from '@/components/ui/count-up'
import { pickFleetStats } from './task-stats-parse'
import { formatBytes, summarizeHardware } from './hardware-summary'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function RealtimeStatusSection() {
  const { t } = useTranslation()
  const edges = useQuery({ queryKey: queryKeys.edges.all, queryFn: listEdges })
  const presence = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })
  const fleet = useQuery({
    queryKey: queryKeys.stats.fleet,
    queryFn: listFleetStats,
  })

  const edgeList = edges.data ?? []
  const enabled = edgeList.filter((e) => e.enabled).length
  const hw = summarizeHardware(edgeList)
  const presenceList = presence.data ?? []
  const online = presenceList.filter((p) => p.edge_online).length
  const comfyRunning = presenceList.filter((p) => p.comfy_running).length
  const fleetData = pickFleetStats(fleet.data)

  const chartData = (fleetData?.nodes ?? []).map((n) => ({
    edge_id: n.edge_id,
    cpu: n.cpu_usage_percent,
    mem: n.mem_usage_percent,
    gpu: n.gpu_usage_percent ?? null,
  }))
  const config: ChartConfig = {
    cpu: { label: t('dashboard.avgCpu'), color: 'var(--primary)' },
    mem: {
      label: t('dashboard.avgMem'),
      color: 'color-mix(in oklch, var(--primary) 55%, var(--background))',
    },
    gpu: {
      label: t('dashboard.avgGpu'),
      color: 'color-mix(in oklch, var(--primary) 30%, var(--background))',
    },
  }

  return (
    <div className='space-y-4'>
      <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
        <Card data-testid='realtime-edges-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.edgesTitle')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {edges.isLoading ? (
              <LoadingSkeleton rows={2} />
            ) : edges.isError ? (
              <ErrorBanner message={errorMessage(edges.error)} onRetry={() => void edges.refetch()} />
            ) : (
              <>
                <div className='text-2xl font-bold'>
                  <CountUp value={enabled} />
                  <span className='text-base font-normal text-muted-foreground'>
                    {' '}/ {edgeList.length}
                  </span>
                </div>
                <div className='mt-2 h-2 w-full overflow-hidden rounded-full bg-muted'>
                  <div
                    className='h-full rounded-full bg-emerald-500'
                    style={{
                      width: edgeList.length
                        ? `${(enabled / edgeList.length) * 100}%`
                        : '0%',
                    }}
                  />
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card data-testid='realtime-presence-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.onlineStatus')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {presence.isLoading ? (
              <LoadingSkeleton rows={2} />
            ) : presence.isError ? (
              <ErrorBanner message={errorMessage(presence.error)} onRetry={() => void presence.refetch()} />
            ) : (
              <>
                <div className='text-2xl font-bold'>
                  <CountUp value={online} />
                  <span className='text-base font-normal text-muted-foreground'>
                    {' '}/ {edgeList.length} 在线
                  </span>
                </div>
                <p className='text-sm text-muted-foreground'>
                  {t('dashboard.comfyRunning')}:{' '}
                  <CountUp value={comfyRunning} />
                </p>
              </>
            )}
          </CardContent>
        </Card>

        <Card data-testid='realtime-pool-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.computePool')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {edges.isLoading ? (
              <LoadingSkeleton rows={2} />
            ) : edges.isError ? (
              <ErrorBanner message={errorMessage(edges.error)} onRetry={() => void edges.refetch()} />
            ) : (
              <>
                <div className='text-2xl font-bold'>
                  <CountUp value={hw.gpuCount} />
                  <span className='text-base font-normal text-muted-foreground'>
                    {' '}GPU · {formatBytes(hw.vramBytes)}
                  </span>
                </div>
                <p className='text-sm text-muted-foreground'>
                  {hw.cpuCores} {t('dashboard.cores')} · {formatBytes(hw.ramBytes)}{' '}
                  {t('dashboard.memory')}
                </p>
              </>
            )}
          </CardContent>
        </Card>

        <Card data-testid='realtime-load-card'>
          <CardHeader className='pb-2'>
            <CardTitle className='text-sm font-medium'>
              {t('dashboard.liveLoadSummary')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {fleet.isLoading ? (
              <LoadingSkeleton rows={3} />
            ) : fleet.isError ? (
              <ErrorBanner message={errorMessage(fleet.error)} onRetry={() => void fleet.refetch()} />
            ) : fleetData ? (
              <>
                <div className='flex justify-between text-sm'>
                  <span>{t('dashboard.avgCpu')}</span>
                  <b className='tabular-nums'>
                    <CountUp
                      value={fleetData.avg_cpu_usage_percent}
                      format={(v) => `${Math.round(v)}%`}
                    />
                  </b>
                </div>
                <div className='flex justify-between text-sm'>
                  <span>{t('dashboard.avgGpu')}</span>
                  <b className='tabular-nums'>
                    {fleetData.avg_gpu_usage_percent == null
                      ? '—'
                      : (
                          <CountUp
                            value={fleetData.avg_gpu_usage_percent}
                            format={(v) => `${Math.round(v)}%`}
                          />
                        )}
                  </b>
                </div>
                <div className='flex justify-between text-sm'>
                  <span>{t('dashboard.vramUsage')}</span>
                  <b className='tabular-nums'>
                    {formatBytes(fleetData.vram_used_bytes)} /{' '}
                    {formatBytes(fleetData.vram_total_bytes)}
                  </b>
                </div>
                <div className='flex justify-between text-sm'>
                  <span>{t('dashboard.hottestNode')}</span>
                  <b className='tabular-nums'>
                    {fleetData.hottest
                      ? `${fleetData.hottest.edge_id} (${fleetData.hottest.cpu_usage_percent.toFixed(0)}%)`
                      : '—'}
                  </b>
                </div>
              </>
            ) : (
              <p className='text-sm text-muted-foreground'>{t('common.empty')}</p>
            )}
          </CardContent>
        </Card>
      </div>

      <Card data-testid='realtime-fleet-chart-card'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-base font-semibold'>
            {t('dashboard.fleetTitle')}
          </CardTitle>
          <CardDescription>{t('dashboard.fleetHint')}</CardDescription>
        </CardHeader>
        <CardContent>
          {fleet.isLoading ? (
            <LoadingSkeleton rows={4} />
          ) : fleet.isError ? (
            <ErrorBanner message={errorMessage(fleet.error)} onRetry={() => void fleet.refetch()} />
          ) : (
            <ChartContainer
              config={config}
              className='h-[220px] w-full min-w-0 sm:h-[260px]'
            >
              <BarChart data={chartData} margin={{ top: 8, right: 4, bottom: 0, left: 4 }}>
                <CartesianGrid vertical={false} />
                <XAxis dataKey='edge_id' tickLine={false} axisLine={false} tickMargin={4} />
                <YAxis tickLine={false} axisLine={false} width={36} unit='%' />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Bar dataKey='cpu' fill='var(--color-cpu)' radius={2} />
                <Bar dataKey='mem' fill='var(--color-mem)' radius={2} />
                <Bar dataKey='gpu' fill='var(--color-gpu)' radius={2} />
              </BarChart>
            </ChartContainer>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

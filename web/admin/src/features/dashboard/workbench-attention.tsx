import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { useQuery } from '@tanstack/react-query'
import { listCases } from '@/lib/api/cases'
import { listEdges, listPresence } from '@/lib/api/edges'
import { listTaskErrorStats } from '@/lib/api/stats'
import { queryKeys } from '@/lib/api/query-keys'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { pickErrorItems } from './task-stats-parse'
import type { StatsRange } from './task-range-picker'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function WorkbenchAttention({ range }: { range: StatsRange }) {
  const { t } = useTranslation()
  const edges = useQuery({ queryKey: queryKeys.edges.all, queryFn: listEdges })
  const presence = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })
  const cases = useQuery({
    queryKey: queryKeys.cases.all,
    queryFn: () => listCases({ enabled: false }),
  })
  const errors = useQuery({
    queryKey: queryKeys.stats.tasksErrors(range.from, range.to),
    queryFn: () => listTaskErrorStats({ ...range, limit: 5 }),
  })

  const edgeList = edges.data ?? []
  const presenceList = presence.data ?? []
  const onlineIds = new Set(presenceList.filter((p) => p.edge_online).map((p) => p.id))
  const comfyIds = new Set(presenceList.filter((p) => p.comfy_running).map((p) => p.id))
  const offline = edgeList.filter((e) => !(onlineIds.has(e.id) && comfyIds.has(e.id)))
  const disabledCases = (cases.data ?? []).filter((c) => !c.enabled)
  const errorItems = pickErrorItems(errors.data)

  return (
    <div data-testid='workbench-attention' className='space-y-4'>
      <Card data-testid='workbench-node-abnormal'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.workbench.nodeAbnormalTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {edges.isLoading || presence.isLoading ? (
            <LoadingSkeleton rows={3} />
          ) : edges.isError || presence.isError ? (
            <ErrorBanner
              message={errorMessage(edges.error ?? presence.error)}
              onRetry={() => void (edges.refetch(), presence.refetch())}
            />
          ) : offline.length ? (
            <ul className='space-y-1 text-sm'>
              {offline.map((e) => {
                const online = onlineIds.has(e.id)
                const comfy = comfyIds.has(e.id)
                return (
                  <li key={e.id} className='flex items-center gap-2'>
                    <span className='size-2 rounded-full bg-destructive' />
                    <Link to='/edges' className='truncate'>
                      {e.name || e.id}
                    </Link>
                    <span className='ms-auto text-xs text-muted-foreground'>
                      {!online
                        ? t('dashboard.workbench.offline')
                        : !comfy
                        ? t('dashboard.workbench.comfyNotRunning')
                        : ''}
                    </span>
                  </li>
                )
              })}
            </ul>
          ) : (
            <p className='text-sm text-muted-foreground'>
              {t('dashboard.workbench.allHealthy')}
            </p>
          )}
        </CardContent>
      </Card>

      <Card data-testid='workbench-disabled-workflow'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.workbench.disabledWorkflowTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {cases.isLoading ? (
            <LoadingSkeleton rows={3} />
          ) : cases.isError ? (
            <ErrorBanner message={errorMessage(cases.error)} onRetry={() => void cases.refetch()} />
          ) : disabledCases.length ? (
            <ul className='space-y-1 text-sm'>
              {disabledCases.map((c) => (
                <li key={c.id} className='flex items-center gap-2'>
                  <span className='size-2 rounded-full bg-muted-foreground' />
                  <Link to='/cases' className='truncate'>
                    {c.name}
                  </Link>
                </li>
              ))}
            </ul>
          ) : (
            <p className='text-sm text-muted-foreground'>
              {t('dashboard.workbench.allEnabled')}
            </p>
          )}
        </CardContent>
      </Card>

      <Card data-testid='workbench-failed-task'>
        <CardHeader className='pb-2'>
          <CardTitle className='text-sm font-medium'>
            {t('dashboard.workbench.failedTaskTopTitle')}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {errors.isLoading ? (
            <LoadingSkeleton rows={3} />
          ) : errors.isError ? (
            <ErrorBanner message={errorMessage(errors.error)} onRetry={() => void errors.refetch()} />
          ) : errorItems.length ? (
            <ul className='space-y-1 text-sm'>
              {errorItems.map((e) => (
                <li key={e.error_code} className='flex items-center justify-between gap-2'>
                  <span className='truncate'>{e.error_code}</span>
                  <span className='tabular-nums'>{e.count}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className='text-sm text-muted-foreground'>
              {t('dashboard.workbench.noFailedTasks')}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

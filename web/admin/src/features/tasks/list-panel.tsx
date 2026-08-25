import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Button } from '@/components/ui/button'
import type { TaskRecord } from '@/lib/api/types'

export const TASK_STATUS_FILTERS = [
  'all',
  'pending',
  'queued',
  'running',
  'succeeded',
  'failed',
  'cancelled',
] as const

export type TaskStatusFilter = (typeof TASK_STATUS_FILTERS)[number]

const TASK_STATUS_LABEL_KEYS: Record<
  Exclude<TaskStatusFilter, 'all'>,
  string
> = {
  pending: 'tasks.statusPending',
  queued: 'tasks.statusQueued',
  running: 'tasks.statusRunning',
  succeeded: 'tasks.statusSucceeded',
  failed: 'tasks.statusFailed',
  cancelled: 'tasks.statusCancelled',
}

export function taskStatusLabelKey(
  status: string,
): string | undefined {
  if (status in TASK_STATUS_LABEL_KEYS) {
    return TASK_STATUS_LABEL_KEYS[status as Exclude<TaskStatusFilter, 'all'>]
  }
  return undefined
}

type Props = {
  items: TaskRecord[]
  onOpenDetail: (item: TaskRecord) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function TaskListPanel({
  items,
  onOpenDetail,
  isLoading,
  isError,
  errorMessage,
  onRetry,
}: Props) {
  const { t } = useTranslation()

  return (
    <div data-testid='tasks-list-panel'>
      {isError ? (
        <div className='p-4'>
          <ErrorBanner message={errorMessage} onRetry={onRetry} />
        </div>
      ) : null}

      {isLoading ? (
        <div className='p-4'>
          <LoadingSkeleton rows={5} />
        </div>
      ) : null}

      {!isLoading && !isError ? (
        <>
          <div className='hidden overflow-x-auto pt-3 md:block'>
            <div className='relative w-full overflow-auto'>
              <table className='w-full caption-bottom text-sm min-w-[980px]'>
                <thead className='[&_tr]:border-b'>
                  <tr className='group/row border-b bg-muted/40 transition-colors hover:bg-muted/40'>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('tasks.fieldId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('tasks.fieldStatus')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('tasks.fieldCaseId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('tasks.fieldSessionId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('tasks.fieldEdgeId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('tasks.fieldCreatedAt')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium w-[48px]' />
                  </tr>
                </thead>
                <tbody className='[&_tr:last-child]:border-0'>
                  {items.length === 0 ? (
                    <tr>
                      <td colSpan={7} className='p-0'>
                        <EmptyState
                          className='py-8'
                          message={t('tasks.empty')}
                        />
                      </td>
                    </tr>
                  ) : (
                    items.map((item) => {
                      const statusKey = taskStatusLabelKey(item.status)
                      return (
                        <tr
                          key={item.id}
                          className='group/row border-b transition-colors hover:bg-muted/50'
                        >
                          <td className='p-2 align-middle font-medium'>
                            {item.id}
                          </td>
                          <td className='p-2 align-middle'>
                            {statusKey ? t(statusKey) : item.status}
                          </td>
                          <td className='text-muted-foreground p-2 align-middle'>
                            {item.case_id}
                          </td>
                          <td className='text-muted-foreground p-2 align-middle'>
                            {item.session_id}
                          </td>
                          <td className='text-muted-foreground p-2 align-middle'>
                            {item.edge_id || '—'}
                          </td>
                          <td className='text-muted-foreground p-2 align-middle'>
                            {item.created_at}
                          </td>
                          <td className='p-2 align-middle'>
                            <Button
                              type='button'
                              variant='outline'
                              size='sm'
                              onClick={() => onOpenDetail(item)}
                            >
                              {t('common.detail')}
                            </Button>
                          </td>
                        </tr>
                      )
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <div className='flex flex-col gap-4 pt-3 md:hidden'>
            {items.length === 0 ? (
              <EmptyState message={t('tasks.empty')} />
            ) : (
              items.map((item) => {
                const statusKey = taskStatusLabelKey(item.status)
                return (
                  <article
                    key={item.id}
                    className='border-border/70 bg-card rounded-2xl border p-4'
                  >
                    <div className='flex items-start justify-between gap-3'>
                      <div className='min-w-0'>
                        <p className='truncate text-base font-medium'>
                          {item.id}
                        </p>
                        <p className='text-muted-foreground mt-1 text-sm'>
                          {statusKey ? t(statusKey) : item.status}
                        </p>
                      </div>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        className='shrink-0'
                        onClick={() => onOpenDetail(item)}
                      >
                        {t('common.detail')}
                      </Button>
                    </div>
                    <div className='mt-3 grid grid-cols-2 gap-3 text-sm'>
                      <div className='flex flex-col gap-1'>
                        <span className='text-muted-foreground text-xs'>
                          {t('tasks.fieldCaseId')}
                        </span>
                        <span>{item.case_id}</span>
                      </div>
                      <div className='flex flex-col gap-1'>
                        <span className='text-muted-foreground text-xs'>
                          {t('tasks.fieldCreatedAt')}
                        </span>
                        <span>{item.created_at}</span>
                      </div>
                    </div>
                  </article>
                )
              })
            )}
          </div>
        </>
      ) : null}
    </div>
  )
}

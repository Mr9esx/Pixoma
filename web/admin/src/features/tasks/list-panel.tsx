import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { TaskRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'

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

export type TaskListFilters = {
  status: TaskStatusFilter
  q: string
}

type Props = {
  items: TaskRecord[]
  selectedId?: string
  filters: TaskListFilters
  onFiltersChange: (next: TaskListFilters) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function TaskListPanel({
  items,
  selectedId,
  filters,
  onFiltersChange,
  isLoading,
  isError,
  errorMessage,
  onRetry,
}: Props) {
  const { t } = useTranslation()

  return (
    <div className='flex h-full min-h-0 flex-col' data-testid='tasks-list-panel'>
      <div className='border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('tasks.title')}</h2>
      </div>

      <div className='space-y-3 border-b px-4 py-3'>
        <div className='space-y-1'>
          <Label className='text-xs'>{t('tasks.filterStatus')}</Label>
          <Select
            value={filters.status}
            onValueChange={(value: TaskStatusFilter) =>
              onFiltersChange({ ...filters, status: value })
            }
          >
            <SelectTrigger className='w-full' size='sm'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TASK_STATUS_FILTERS.map((status) => (
                <SelectItem key={status} value={status}>
                  {status === 'all'
                    ? t('tasks.filterStatusAll')
                    : t(TASK_STATUS_LABEL_KEYS[status])}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className='space-y-1'>
          <Label htmlFor='tasks-filter-q' className='text-xs'>
            {t('tasks.filterQ')}
          </Label>
          <Input
            id='tasks-filter-q'
            value={filters.q}
            onChange={(e) =>
              onFiltersChange({ ...filters, q: e.target.value })
            }
            placeholder={t('tasks.filterQPlaceholder')}
            autoComplete='off'
          />
        </div>
      </div>

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

      {!isLoading && !isError && items.length === 0 ? (
        <EmptyState message={t('tasks.empty')} />
      ) : null}

      {!isLoading && !isError && items.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {items.map((item) => {
            const selected = selectedId === item.id
            const statusKey = taskStatusLabelKey(item.status)
            return (
              <li key={item.id}>
                <Link
                  to='/tasks/$taskId'
                  params={{ taskId: item.id }}
                  className={cn(
                    'block w-full px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'bg-accent',
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='font-medium'>{item.id}</span>
                    <span className='text-muted-foreground shrink-0 text-xs'>
                      {statusKey ? t(statusKey) : item.status}
                    </span>
                  </div>
                  <div className='text-muted-foreground mt-0.5 truncate text-xs'>
                    {t('tasks.fieldCaseId')}: {item.case_id}
                  </div>
                </Link>
              </li>
            )
          })}
        </ul>
      ) : null}
    </div>
  )
}

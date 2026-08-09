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
import type { SessionRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'

export const SESSION_STATUS_FILTERS = [
  'all',
  'collecting',
  'confirming',
  'submitted',
  'exited',
] as const

export type SessionStatusFilter = (typeof SESSION_STATUS_FILTERS)[number]

export type SessionListFilters = {
  user_id: string
  status: SessionStatusFilter
  q: string
}

type Props = {
  items: SessionRecord[]
  selectedId?: string
  filters: SessionListFilters
  onFiltersChange: (next: SessionListFilters) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function SessionListPanel({
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
    <div
      className='flex h-full min-h-0 flex-col'
      data-testid='sessions-list-panel'
    >
      <div className='border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('sessions.title')}</h2>
      </div>

      <div className='space-y-3 border-b px-4 py-3'>
        <div className='space-y-1'>
          <Label htmlFor='sessions-filter-user' className='text-xs'>
            {t('sessions.filterUserId')}
          </Label>
          <Input
            id='sessions-filter-user'
            value={filters.user_id}
            onChange={(e) =>
              onFiltersChange({ ...filters, user_id: e.target.value })
            }
            placeholder={t('sessions.filterUserIdPlaceholder')}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1'>
          <Label className='text-xs'>{t('sessions.filterStatus')}</Label>
          <Select
            value={filters.status}
            onValueChange={(value: SessionStatusFilter) =>
              onFiltersChange({ ...filters, status: value })
            }
          >
            <SelectTrigger className='w-full' size='sm'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SESSION_STATUS_FILTERS.map((status) => (
                <SelectItem key={status} value={status}>
                  {status === 'all'
                    ? t('sessions.filterStatusAll')
                    : status}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className='space-y-1'>
          <Label htmlFor='sessions-filter-q' className='text-xs'>
            {t('sessions.filterQ')}
          </Label>
          <Input
            id='sessions-filter-q'
            value={filters.q}
            onChange={(e) =>
              onFiltersChange({ ...filters, q: e.target.value })
            }
            placeholder={t('sessions.filterQPlaceholder')}
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
        <EmptyState message={t('sessions.empty')} />
      ) : null}

      {!isLoading && !isError && items.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {items.map((item) => {
            const selected = selectedId === item.id
            return (
              <li key={item.id}>
                <Link
                  to='/sessions/$sessionId'
                  params={{ sessionId: item.id }}
                  className={cn(
                    'block w-full px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'bg-accent',
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='font-medium'>{item.id}</span>
                    <span className='text-muted-foreground shrink-0 text-xs'>
                      {item.status}
                    </span>
                  </div>
                  <div className='text-muted-foreground mt-0.5 truncate text-xs'>
                    {t('sessions.fieldUserId')}: {item.user_id}
                  </div>
                  <div className='text-muted-foreground truncate text-xs'>
                    {t('sessions.fieldCaseId')}: {item.case_id}
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

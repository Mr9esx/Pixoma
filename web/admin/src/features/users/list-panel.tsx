import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { UserRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'

export type UserListFilters = {
  q: string
  tg_user_id: string
}

type Props = {
  items: UserRecord[]
  selectedId?: string
  filters: UserListFilters
  onFiltersChange: (next: UserListFilters) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function UserListPanel({
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
    <div className='flex h-full min-h-0 flex-col' data-testid='users-list-panel'>
      <div className='border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('users.title')}</h2>
      </div>

      <div className='space-y-3 border-b px-4 py-3'>
        <div className='space-y-1'>
          <Label htmlFor='users-filter-q' className='text-xs'>
            {t('users.filterQ')}
          </Label>
          <Input
            id='users-filter-q'
            value={filters.q}
            onChange={(e) =>
              onFiltersChange({ ...filters, q: e.target.value })
            }
            placeholder={t('users.filterQPlaceholder')}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1'>
          <Label htmlFor='users-filter-tg' className='text-xs'>
            {t('users.filterTgUserId')}
          </Label>
          <Input
            id='users-filter-tg'
            value={filters.tg_user_id}
            onChange={(e) =>
              onFiltersChange({ ...filters, tg_user_id: e.target.value })
            }
            placeholder={t('users.filterTgUserIdPlaceholder')}
            inputMode='numeric'
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
        <EmptyState message={t('users.empty')} />
      ) : null}

      {!isLoading && !isError && items.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {items.map((item) => {
            const selected = selectedId === item.id
            return (
              <li key={item.id}>
                <Link
                  to='/users/$userId'
                  params={{ userId: item.id }}
                  className={cn(
                    'block w-full px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'bg-accent',
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='font-medium'>{item.id}</span>
                    <span className='text-muted-foreground shrink-0 text-xs'>
                      {item.tg_user_id}
                    </span>
                  </div>
                  <div className='text-muted-foreground mt-0.5 truncate text-xs'>
                    {item.username || '—'}
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

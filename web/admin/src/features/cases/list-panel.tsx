import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { FilterSegment } from '@/components/filters/filter-segment'

export type CaseListFilters = {
  q: string
  enabled: 'all' | 'true' | 'false'
}

type Props = {
  items: CaseRecord[]
  selectedId?: number
  filters: CaseListFilters
  onFiltersChange: (next: CaseListFilters) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function CaseListPanel({
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
      data-testid='cases-list-panel'
    >
      <div className='space-y-2 border-b px-4 py-3'>
        <Input
          id='cases-filter-q'
          value={filters.q}
          onChange={(e) => onFiltersChange({ ...filters, q: e.target.value })}
          placeholder={t('cases.filterQPlaceholder')}
          autoComplete='off'
          aria-label={t('cases.filterQ')}
        />
        <FilterSegment
          data-testid='cases-filter-enabled'
          aria-label={t('cases.filterEnabled')}
          value={filters.enabled}
          onValueChange={(enabled) => onFiltersChange({ ...filters, enabled })}
          options={[
            { value: 'all', label: t('cases.filterEnabledAll') },
            { value: 'true', label: t('cases.filterEnabledTrue') },
            { value: 'false', label: t('cases.filterEnabledFalse') },
          ]}
        />
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
        <EmptyState message={t('cases.empty')} />
      ) : null}

      {!isLoading && !isError && items.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {items.map((item) => {
            const selected = selectedId === item.id
            return (
              <li key={item.id}>
                <Link
                  to='/cases/$caseId'
                  params={{ caseId: String(item.id) }}
                  className={cn(
                    'block w-full px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'bg-accent'
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='font-medium'>{item.id}</span>
                    <span
                      className={cn(
                        'shrink-0 text-xs',
                        item.enabled
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : 'text-muted-foreground'
                      )}
                    >
                      {item.enabled ? t('cases.enabled') : t('cases.disabled')}
                    </span>
                  </div>
                  <div className='mt-0.5 truncate text-xs text-muted-foreground'>
                    {item.name}
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

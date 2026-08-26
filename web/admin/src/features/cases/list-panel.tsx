import { Link } from '@tanstack/react-router'
import { Boxes } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { LongText } from '@/components/long-text'

export type CaseListFilters = {
  q: string
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
      id='case-list'
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
        <Empty>
          <EmptyHeader className='max-w-none'>
            <EmptyMedia variant='icon'>
              <Boxes />
            </EmptyMedia>
            <EmptyTitle className='text-sm font-medium'>
              {t('cases.empty')}
            </EmptyTitle>
          </EmptyHeader>
        </Empty>
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
                  <span className='block truncate font-medium'>
                    {item.name}
                  </span>
                  {item.description ? (
                    <LongText className='mt-0.5 text-xs text-muted-foreground'>
                      {item.description}
                    </LongText>
                  ) : null}
                </Link>
              </li>
            )
          })}
        </ul>
      ) : null}
    </div>
  )
}

import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { CaseRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Empty, EmptyDescription } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { LongText } from '@/components/long-text'
import { StatusDot } from '@/components/status-dot'
import type { EntityHealth } from '@/features/link-health/lib/references'

export type CaseListFilters = {
  q: string
}

type Props = {
  items: CaseRecord[]
  healthByCase?: Record<number, EntityHealth>
  healthReady?: boolean
  selectedId?: number
  filters: CaseListFilters
  onFiltersChange: (next: CaseListFilters) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
  footer?: ReactNode
}

export function CaseListPanel({
  items,
  healthByCase,
  healthReady = false,
  selectedId,
  filters,
  onFiltersChange,
  isLoading,
  isError,
  errorMessage,
  onRetry,
  footer,
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
          <EmptyDescription>{t('cases.noData')}</EmptyDescription>
        </Empty>
      ) : null}

      {!isLoading && !isError && items.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {items.map((item) => {
            const selected = selectedId === item.id
            const health = healthReady ? healthByCase?.[item.id] : undefined
            const problems =
              (item.enabled ? 0 : 1) + (health?.breakpoints.length ?? 1)
            return (
              <li key={item.id}>
                <Link
                  to='/cases/$caseId'
                  params={{ caseId: String(item.id) }}
                  className={cn(
                    'block w-full border-l-2 border-l-transparent px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'border-l-foreground bg-accent'
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='truncate font-medium'>{item.name}</span>
                    <StatusDot
                      problems={problems}
                      label={
                        problems === 0 ? t('cases.enabled') : t('status.issue')
                      }
                    />
                  </div>
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

      {items.length > 0 && footer ? (
        <footer className='mt-auto border-t px-4 py-3'>{footer}</footer>
      ) : null}
    </div>
  )
}

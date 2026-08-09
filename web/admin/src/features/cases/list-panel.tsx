import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { CaseRecord } from '@/lib/api/types'
import { cn } from '@/lib/utils'

export type CaseListFilters = {
  q: string
  menuKey: string
  enabled: 'all' | 'true' | 'false'
}

type Props = {
  items: CaseRecord[]
  selectedId?: string
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
    <div className='flex h-full min-h-0 flex-col' data-testid='cases-list-panel'>
      <div className='flex items-center justify-between gap-2 border-b px-4 py-3'>
        <h2 className='text-sm font-semibold'>{t('cases.title')}</h2>
        <Button asChild size='sm'>
          <Link to='/cases/$caseId' params={{ caseId: 'new' }}>
            {t('common.create')}
          </Link>
        </Button>
      </div>

      <div className='space-y-3 border-b px-4 py-3'>
        <div className='space-y-1'>
          <Label htmlFor='cases-filter-q' className='text-xs'>
            {t('cases.filterQ')}
          </Label>
          <Input
            id='cases-filter-q'
            value={filters.q}
            onChange={(e) =>
              onFiltersChange({ ...filters, q: e.target.value })
            }
            placeholder={t('cases.filterQPlaceholder')}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1'>
          <Label htmlFor='cases-filter-menu-key' className='text-xs'>
            {t('cases.filterMenuKey')}
          </Label>
          <Input
            id='cases-filter-menu-key'
            value={filters.menuKey}
            onChange={(e) =>
              onFiltersChange({ ...filters, menuKey: e.target.value })
            }
            placeholder={t('cases.filterMenuKeyPlaceholder')}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1'>
          <Label className='text-xs'>{t('cases.filterEnabled')}</Label>
          <Select
            value={filters.enabled}
            onValueChange={(value: CaseListFilters['enabled']) =>
              onFiltersChange({ ...filters, enabled: value })
            }
          >
            <SelectTrigger className='w-full' size='sm'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='all'>{t('cases.filterEnabledAll')}</SelectItem>
              <SelectItem value='true'>
                {t('cases.filterEnabledTrue')}
              </SelectItem>
              <SelectItem value='false'>
                {t('cases.filterEnabledFalse')}
              </SelectItem>
            </SelectContent>
          </Select>
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
                  params={{ caseId: item.id }}
                  className={cn(
                    'block w-full px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'bg-accent',
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='font-medium'>{item.id}</span>
                    <span
                      className={cn(
                        'shrink-0 text-xs',
                        item.enabled
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : 'text-muted-foreground',
                      )}
                    >
                      {item.enabled
                        ? t('cases.enabled')
                        : t('cases.disabled')}
                    </span>
                  </div>
                  <div className='text-muted-foreground mt-0.5 truncate text-xs'>
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

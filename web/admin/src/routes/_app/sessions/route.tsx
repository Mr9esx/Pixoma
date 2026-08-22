import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { Search } from 'lucide-react'
import { cn } from '@/lib/utils'
import { queryKeys } from '@/lib/api/query-keys'
import { listSessions } from '@/lib/api/sessions'
import type { SessionRecord } from '@/lib/api/types'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { SessionDetailPanel } from '@/features/sessions/detail-panel'
import {
  SessionListPanel,
  SESSION_STATUS_FILTERS,
  sessionStatusLabelKey,
  type SessionStatusFilter,
} from '@/features/sessions/list-panel'

export const Route = createFileRoute('/_app/sessions')({
  component: SessionsLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function SessionsLayout() {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<SessionRecord | null>(null)
  const [filters, setFilters] = useState<{
    status: SessionStatusFilter
    q: string
  }>({
    status: 'all',
    q: '',
  })

  const listParams = {
    status: filters.status === 'all' ? undefined : filters.status,
    q: filters.q.trim() || undefined,
  }

  const listQuery = useQuery({
    queryKey: [...queryKeys.sessions.all, listParams] as const,
    queryFn: () => listSessions(listParams),
  })
  const items = listQuery.data ?? []

  return (
    <div
      className='min-h-0 flex-1 overflow-auto has-[&>[data-layout=fixed]]:flex has-[&>[data-layout=fixed]]:overflow-hidden'
      data-testid='sessions-page'
    >
      <div className='flex flex-col'>
        <div className='flex flex-col'>
          <div className='hidden gap-4 pb-8 md:flex lg:flex-row lg:items-start lg:justify-between'>
            <div className='flex flex-col gap-1'>
              <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
                {t('sessions.title')}
              </h1>
              <p className='text-muted-foreground text-sm'>
                {t('sessions.description')}
              </p>
            </div>
          </div>

          <div className='flex flex-col gap-2 pb-6 md:hidden'>
            <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
              {t('sessions.title')}
            </h1>
            <p className='text-muted-foreground text-sm'>
              {t('sessions.description')}
            </p>
          </div>

          <div className='pt-2 pb-1'>
            <div className='border-b'>
              <div
                role='tablist'
                aria-orientation='horizontal'
                className='text-muted-foreground inline-flex h-auto items-center justify-center gap-4 rounded-lg bg-transparent p-0'
              >
                {SESSION_STATUS_FILTERS.map((status) => {
                  const active = filters.status === status
                  return (
                    <button
                      key={status}
                      type='button'
                      role='tab'
                      aria-selected={active}
                      onClick={() =>
                        setFilters((prev) => ({ ...prev, status }))
                      }
                      className={cn(
                        'inline-flex items-center justify-center rounded-none border-b-2 border-transparent px-1 pt-0 pb-3 text-sm font-medium whitespace-nowrap transition-all',
                        active && 'text-foreground border-foreground'
                      )}
                    >
                      {status === 'all'
                        ? t('sessions.filterStatusAll')
                        : t(sessionStatusLabelKey(status) ?? status)}
                    </button>
                  )
                })}
              </div>
            </div>
          </div>

          <div className='flex flex-col gap-4 pt-2 md:flex-row md:items-center md:justify-between'>
            <div className='flex flex-wrap items-center gap-3'>
              <div className='relative w-full sm:w-72'>
                <Search className='text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2' />
                <Input
                  id='sessions-filter-q'
                  type='search'
                  value={filters.q}
                  onChange={(e) =>
                    setFilters((prev) => ({ ...prev, q: e.target.value }))
                  }
                  placeholder={t('sessions.filterQPlaceholder')}
                  aria-label={t('sessions.filterQ')}
                  autoComplete='off'
                  className='border-input w-full bg-transparent pl-9 shadow-xs'
                />
              </div>
            </div>
          </div>

          <SessionListPanel
            items={items}
            onOpenDetail={setDetail}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        </div>
      </div>

      <Dialog
        open={detail !== null}
        onOpenChange={(open) => {
          if (!open) setDetail(null)
        }}
      >
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>{t('sessions.detailHeading')}</DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto px-5 py-4'>
            {detail ? <SessionDetailPanel id={detail.id} /> : null}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { Channel } from '@/lib/api/channels'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'

type Props = {
  items: Channel[]
  selectedId?: string
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function ChannelListPanel({
  items,
  selectedId,
  isLoading,
  isError,
  errorMessage,
  onRetry,
}: Props) {
  const { t } = useTranslation()
  const [q, setQ] = useState('')
  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase()
    if (!needle) return items
    return items.filter(
      (ch) =>
        ch.name.toLowerCase().includes(needle) ||
        ch.platform.toLowerCase().includes(needle)
    )
  }, [items, q])

  return (
    <div
      className='flex h-full min-h-0 flex-col'
      data-testid='channels-list-panel'
    >
      <div className='space-y-2 border-b px-4 py-3'>
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t('channels.listSearch')}
          autoComplete='off'
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

      {!isLoading && !isError && filtered.length === 0 ? (
        <EmptyState message={t('channels.empty')} />
      ) : null}

      {!isLoading && !isError && filtered.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {filtered.map((ch) => {
            const selected = selectedId === ch.id
            return (
              <li key={ch.id}>
                <Link
                  to='/channels/$id'
                  params={{ id: ch.id }}
                  className={cn(
                    'block w-full px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'bg-accent'
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='truncate font-medium'>{ch.name}</span>
                    <span
                      className={cn(
                        'shrink-0 rounded-sm px-1.5 py-0.5 text-[10px]',
                        ch.enabled
                          ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
                          : 'bg-muted text-muted-foreground'
                      )}
                    >
                      {ch.enabled
                        ? t('channels.enabled')
                        : t('channels.disabled')}
                    </span>
                  </div>
                  <div className='mt-0.5 truncate text-xs text-muted-foreground'>
                    {ch.platform} · {ch.token_masked}
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

import { useMemo, useState, type ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { Channel } from '@/lib/api/channels'
import { cn } from '@/lib/utils'
import { Empty, EmptyDescription } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { StatusDot } from '@/components/status-dot'

type Props = {
  items: Channel[]
  selectedId?: string
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
  footer?: ReactNode
}

export function ChannelListPanel({
  items,
  selectedId,
  isLoading,
  isError,
  errorMessage,
  onRetry,
  footer,
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
      id='channel-list'
      className='flex h-full min-h-0 flex-col'
      data-testid='channels-list-panel'
    >
      <div className='flex flex-col gap-2 border-b px-4 py-3'>
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
        <Empty>
          <EmptyDescription>{t('channels.noData')}</EmptyDescription>
        </Empty>
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
                    'block w-full border-l-2 border-l-transparent px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'border-l-foreground bg-accent'
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='truncate font-medium'>{ch.name}</span>
                    <StatusDot
                      problems={ch.enabled ? 0 : 1}
                      label={
                        ch.enabled
                          ? t('channels.enabled')
                          : t('channels.disabled')
                      }
                    />
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

      {items.length > 0 && footer ? (
        <footer className='mt-auto border-t px-4 py-3'>{footer}</footer>
      ) : null}
    </div>
  )
}

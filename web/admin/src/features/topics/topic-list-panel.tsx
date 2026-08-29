import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { Topic } from '@/lib/api/topics'
import { cn } from '@/lib/utils'
import { Empty, EmptyDescription } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'

type Props = {
  items: Topic[]
  selectedKey?: string
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function TopicListPanel({
  items,
  selectedKey,
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
      (topic) =>
        topic.key.toLowerCase().includes(needle) ||
        topic.name.toLowerCase().includes(needle)
    )
  }, [items, q])

  return (
    <div
      className='flex h-full min-h-0 flex-col'
      data-testid='topics-list-panel'
    >
      <div className='space-y-2 border-b px-4 py-3'>
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t('topics.listSearch')}
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
          <EmptyDescription>{t('topics.noData')}</EmptyDescription>
        </Empty>
      ) : null}

      {!isLoading && !isError && filtered.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {filtered.map((topic) => {
            const selected = selectedKey === topic.key
            return (
              <li key={topic.key}>
                <Link
                  to='/topics/$key'
                  params={{ key: topic.key }}
                  className={cn(
                    'block w-full border-l-2 border-l-transparent px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'border-l-foreground bg-accent'
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='truncate font-medium'>{topic.name}</span>
                    <span
                      className={cn(
                        'shrink-0 rounded-sm px-1.5 py-0.5 text-[10px]',
                        topic.enabled
                          ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
                          : 'bg-muted text-muted-foreground'
                      )}
                    >
                      {topic.enabled
                        ? t('topics.enabled')
                        : t('topics.disabled')}
                    </span>
                  </div>
                  <div className='mt-0.5 truncate text-xs text-muted-foreground'>
                    {topic.key}
                    {topic.key === 'default'
                      ? ` · ${t('topics.defaultBadge')}`
                      : ''}
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

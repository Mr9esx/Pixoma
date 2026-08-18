import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import type { ComfyEdge, EdgePresence } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { kit } from './kit-classes'
import { listHealthTone } from './list-health'
import { formatBytes } from './observation'

type Props = {
  items: ComfyEdge[]
  presenceById?: Record<string, EdgePresence>
  selectedId?: string
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function EdgeListPanel({
  items,
  presenceById,
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
    return items.filter((item) =>
      (item.name || '').toLowerCase().includes(needle)
    )
  }, [items, q])

  return (
    <div
      className='flex h-full min-h-0 flex-col'
      data-testid='edges-list-panel'
    >
      <div className='p-3'>
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t('edges.fieldName')}
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
        <EmptyState message={t('edges.empty')} />
      ) : null}

      {!isLoading && !isError && filtered.length > 0 ? (
        <ul className='min-h-0 flex-1 divide-y overflow-auto'>
          {filtered.map((item) => {
            const selected = selectedId === item.id
            const presence = presenceById?.[item.id]
            const tone = listHealthTone({
              enabled: item.enabled,
              edgeOnline: presence?.edge_online === true,
              comfyRunning: presence?.comfy_running === true,
            })
            const cores = item.hardware?.cpu_cores
            const vramBytes = (item.hardware?.gpus ?? []).reduce(
              (sum, gpu) => sum + (gpu.vram_bytes ?? 0),
              0
            )
            const specParts: string[] = []
            if (cores && cores > 0) {
              specParts.push(t('edges.coresValue', { count: cores }))
            }
            if (vramBytes > 0) {
              specParts.push(formatBytes(vramBytes))
            }
            const specLine = specParts.join(' · ')
            return (
              <li key={item.id}>
                <Link
                  to='/edges/$edgeId'
                  params={{ edgeId: item.id }}
                  className={cn(
                    'block w-full px-4 py-3 text-left text-sm hover:bg-accent',
                    selected && 'bg-accent'
                  )}
                >
                  <div className='flex items-center justify-between gap-2'>
                    <span className='truncate font-medium'>
                      {item.name || item.id}
                    </span>
                    <span
                      className={kit.healthDot[tone]}
                      data-health={tone}
                      aria-hidden
                    />
                  </div>
                  {specLine ? (
                    <p className='mt-0.5 truncate text-xs text-muted-foreground'>
                      {specLine}
                    </p>
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

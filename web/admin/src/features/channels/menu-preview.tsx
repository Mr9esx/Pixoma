import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { getChannelMenuPreview } from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'

export function ChannelMenuPreview({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const previewQuery = useQuery({
    queryKey: [...queryKeys.channels.menu(channelId), 'preview'] as const,
    queryFn: () => getChannelMenuPreview(channelId),
  })

  if (previewQuery.isLoading) return <LoadingSkeleton rows={4} />
  if (previewQuery.isError) {
    return <ErrorBanner message={String(previewQuery.error)} />
  }
  const preview = previewQuery.data
  if (!preview) return null

  return (
    <div className='max-w-xl space-y-5'>
      <div>
        <h3 className='mb-2 text-sm font-semibold'>{t('preview.mainKeyboard')}</h3>
        <div className='space-y-2 rounded-md border p-4'>
          {preview.main_keyboard.map((row, i) => (
            <div key={i} className='flex flex-wrap gap-2'>
              {row.map((label) => (
                <span
                  key={label}
                  className='rounded-md border border-cyan-500/40 bg-cyan-500/10 px-3 py-1.5 text-sm'
                >
                  {label}
                </span>
              ))}
            </div>
          ))}
        </div>
      </div>
      {Object.entries(preview.groups).map(([id, g]) => (
        <div key={id}>
          <h3 className='mb-2 text-sm font-semibold'>
            {t('preview.group')}: {g.title}
          </h3>
          <div className='space-y-2 rounded-md border p-4'>
            {g.buttons.map((b, i) => (
              <div key={i} className='text-sm'>
                <span className='mr-2 text-muted-foreground'>
                  {b.kind === 'capability' ? '▶' : '📁'}
                </span>
                {b.label}
              </div>
            ))}
          </div>
        </div>
      ))}
      <p className='text-xs text-muted-foreground'>{t('preview.hint')}</p>
    </div>
  )
}

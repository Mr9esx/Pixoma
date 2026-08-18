import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { listChannels } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'

export const Route = createFileRoute('/_app/channels/')({
  component: ChannelsPage,
})

function ChannelsPage() {
  const { t } = useTranslation()
  const channelsQuery = useQuery({
    queryKey: queryKeys.channels.all,
    queryFn: listChannels,
  })

  if (channelsQuery.isLoading) return <LoadingSkeleton rows={4} />
  if (channelsQuery.isError) {
    return <ErrorBanner message={String(channelsQuery.error)} />
  }
  const channels = channelsQuery.data ?? []

  return (
    <div data-layout='fixed' className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-2xl font-bold tracking-tight'>{t('channels.title')}</h1>
          <p className='text-sm text-muted-foreground'>{t('channels.description')}</p>
        </div>
        <Button asChild>
          <Link to='/channels/new'>{t('channels.new')}</Link>
        </Button>
      </div>
      <div className='min-h-0 flex-1 overflow-auto'>
        {channels.length === 0 ? (
          <Card>
            <CardContent className='py-8 text-center text-sm text-muted-foreground'>
              {t('channels.empty')}
            </CardContent>
          </Card>
        ) : (
          <div className='grid gap-3 md:grid-cols-2'>
            {channels.map((ch) => (
              <Card key={ch.id}>
                <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
                  <CardTitle className='text-base'>{ch.name}</CardTitle>
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs ${
                      ch.enabled ? 'bg-emerald-100 text-emerald-700' : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    {ch.enabled ? t('channels.enabled') : t('channels.disabled')}
                  </span>
                </CardHeader>
                <CardContent className='text-sm text-muted-foreground'>
                  <div>
                    {t('channels.platform')}: {ch.platform}
                  </div>
                  <div>
                    {t('channels.token')}: {ch.token_masked}
                  </div>
                  <Button asChild size='sm' className='mt-3'>
                    <Link to='/channels/$id' params={{ id: ch.id }}>
                      {t('channels.open')}
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

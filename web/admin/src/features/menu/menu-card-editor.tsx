import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { getMenu } from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { MenuPhone } from './menu-phone'

export function MenuCardEditor({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const menuQuery = useQuery({
    queryKey: queryKeys.channels.menu(channelId),
    queryFn: () => getMenu(channelId),
  })

  if (menuQuery.isLoading) {
    return <LoadingSkeleton rows={8} />
  }
  if (menuQuery.isError || !menuQuery.data) {
    return <ErrorBanner message={t('menu.mapLoadFailed')} />
  }

  return (
    <div
      data-testid='menu-card-editor'
      className='mx-auto w-[375px] max-w-full'
    >
      <MenuPhone tree={menuQuery.data} channelId={channelId} />
    </div>
  )
}

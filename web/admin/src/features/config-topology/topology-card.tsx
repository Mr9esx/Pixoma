import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { LinkGraph } from './link-graph'
import { useTopologySource } from './use-topology-source'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function TopologyCard() {
  const { t } = useTranslation()
  const { graphAll, isLoading, isError, error, refetch } = useTopologySource()

  return (
    <Card data-testid='topology-card' className='shadow-none'>
      <CardHeader className='px-4 py-3'>
        <CardTitle className='text-sm'>{t('topology.title')}</CardTitle>
      </CardHeader>
      <CardContent className='px-4 pb-4'>
        <div className='h-[360px]'>
          {isLoading ? (
            <LoadingSkeleton rows={6} />
          ) : isError ? (
            <ErrorBanner
              message={errorMessage(error)}
              onRetry={() => refetch()}
            />
          ) : (
            <LinkGraph graph={graphAll} />
          )}
        </div>
      </CardContent>
    </Card>
  )
}

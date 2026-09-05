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
    <Card
      data-testid='topology-card'
      className='min-w-0 flex-1 gap-3 py-4'
    >
      <CardHeader className='gap-3 px-4'>
        <CardTitle className='text-base font-semibold'>
          {t('topology.title')}
        </CardTitle>
      </CardHeader>
      <CardContent className='grid min-h-0 flex-1 gap-1.5 px-4'>
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

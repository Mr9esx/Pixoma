import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { getCase } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import { CaseForm } from './case-form'
import { MenuPlacementsSection } from './sections/menu-placements'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

type Props = {
  id: string
}

export function CaseDetailPanel({ id }: Props) {
  const { t } = useTranslation()

  const detailQuery = useQuery({
    queryKey: queryKeys.cases.detail(id),
    queryFn: () => getCase(id),
  })

  if (detailQuery.isLoading) {
    return (
      <div data-testid='case-detail-panel'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div data-testid='case-detail-panel' className='space-y-3'>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }

  const record = detailQuery.data
  if (!record) return null

  return (
    <div className='space-y-4' data-testid='case-detail-panel'>
      <div>
        <h2 className='text-lg font-semibold'>{record.id}</h2>
        <p className='text-muted-foreground text-sm'>
          {t('cases.editHeading')}
        </p>
      </div>
      <MenuPlacementsSection caseId={record.id} />
      <CaseForm key={record.id} mode='edit' initial={record} />
    </div>
  )
}

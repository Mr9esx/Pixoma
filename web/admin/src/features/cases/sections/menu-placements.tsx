import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { getCaseMenuPlacements } from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function formatPlacementPath(path: { id: string; label: string }[]): string {
  return path.map((step) => step.label).join(' / ')
}

type Props = {
  caseId: string
  showHeading?: boolean
}

export function MenuPlacementsSection({ caseId, showHeading = true }: Props) {
  const { t } = useTranslation()

  const placementsQuery = useQuery({
    queryKey: queryKeys.cases.menuPlacements(caseId),
    queryFn: () => getCaseMenuPlacements(caseId),
  })

  return (
    <section className='space-y-3' data-testid='case-section-menu-placements'>
      {showHeading ? (
        <h3 className='text-sm font-semibold'>
          {t('cases.menuPlacementsTitle')}
        </h3>
      ) : null}

      {placementsQuery.isError ? (
        <ErrorBanner
          message={errorMessage(placementsQuery.error)}
          onRetry={() => void placementsQuery.refetch()}
        />
      ) : null}

      {placementsQuery.isLoading ? <LoadingSkeleton rows={2} /> : null}

      {!placementsQuery.isLoading && !placementsQuery.isError ? (
        placementsQuery.data?.length ? (
          <ul className='space-y-2'>
            {placementsQuery.data.map((placement) => (
              <li
                key={`${placement.channel_id}:${placement.item_id}`}
                className='flex flex-wrap items-center gap-2 text-sm'
              >
                <span
                  className={`rounded-sm px-1.5 py-0.5 text-[10px] ${
                    placement.kind === 'card_button'
                      ? 'bg-violet-500/15 text-violet-700 dark:text-violet-400'
                      : 'bg-cyan-500/15 text-cyan-700 dark:text-cyan-400'
                  }`}
                >
                  {placement.kind === 'card_button'
                    ? t('cases.placementCardButton')
                    : t('cases.placementMenuItem')}
                </span>
                {formatPlacementPath(placement.path)}
                <span className='text-xs text-muted-foreground'>
                  · {placement.channel_id}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className='text-sm text-muted-foreground'>
            {t('cases.menuPlacementsEmpty')}
          </p>
        )
      ) : null}
    </section>
  )
}

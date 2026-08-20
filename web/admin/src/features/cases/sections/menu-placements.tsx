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
          <div className='overflow-auto rounded-md border'>
            <table className='w-full text-sm'>
              <thead className='bg-muted/40 text-left text-xs text-muted-foreground'>
                <tr>
                  <th className='px-3 py-2 font-medium'>
                    {t('cases.entriesColType')}
                  </th>
                  <th className='px-3 py-2 font-medium'>
                    {t('cases.entriesColEntry')}
                  </th>
                  <th className='px-3 py-2 font-medium'>
                    {t('cases.entriesColChannel')}
                  </th>
                </tr>
              </thead>
              <tbody className='divide-y'>
                {placementsQuery.data.map((placement) => (
                  <tr key={`${placement.channel_id}:${placement.item_id}`}>
                    <td className='px-3 py-2'>
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
                    </td>
                    <td className='px-3 py-2'>
                      {formatPlacementPath(placement.path)}
                    </td>
                    <td className='px-3 py-2 font-mono text-xs text-muted-foreground'>
                      {placement.channel_id}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p className='text-sm text-muted-foreground'>
            {t('cases.menuPlacementsEmpty')}
          </p>
        )
      ) : null}
    </section>
  )
}

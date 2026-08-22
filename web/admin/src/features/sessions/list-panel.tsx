import { useTranslation } from 'react-i18next'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Button } from '@/components/ui/button'
import type { SessionRecord } from '@/lib/api/types'

export const SESSION_STATUS_FILTERS = [
  'all',
  'collecting',
  'confirming',
  'submitted',
  'exited',
] as const

export type SessionStatusFilter = (typeof SESSION_STATUS_FILTERS)[number]

const SESSION_STATUS_LABEL_KEYS: Record<
  Exclude<SessionStatusFilter, 'all'>,
  string
> = {
  collecting: 'sessions.statusCollecting',
  confirming: 'sessions.statusConfirming',
  submitted: 'sessions.statusSubmitted',
  exited: 'sessions.statusExited',
}

export function sessionStatusLabelKey(
  status: string,
): string | undefined {
  if (status in SESSION_STATUS_LABEL_KEYS) {
    return SESSION_STATUS_LABEL_KEYS[
      status as Exclude<SessionStatusFilter, 'all'>
    ]
  }
  return undefined
}

type Props = {
  items: SessionRecord[]
  onOpenDetail: (item: SessionRecord) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function SessionListPanel({
  items,
  onOpenDetail,
  isLoading,
  isError,
  errorMessage,
  onRetry,
}: Props) {
  const { t } = useTranslation()

  return (
    <div data-testid='sessions-list-panel'>
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

      {!isLoading && !isError && items.length === 0 ? (
        <EmptyState message={t('sessions.empty')} />
      ) : null}

      {!isLoading && !isError && items.length > 0 ? (
        <>
          <div className='hidden overflow-x-auto pt-3 md:block'>
            <div className='relative w-full overflow-auto'>
              <table className='w-full caption-bottom text-sm min-w-[980px]'>
                <thead className='[&_tr]:border-b'>
                  <tr className='group/row border-b bg-muted/40 transition-colors hover:bg-muted/40'>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('sessions.fieldId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('sessions.fieldStatus')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('sessions.fieldUserId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('sessions.fieldCaseId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('sessions.fieldChatId')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium'>
                      {t('sessions.fieldCreatedAt')}
                    </th>
                    <th className='text-muted-foreground h-10 px-2 text-left align-middle font-medium w-[48px]' />
                  </tr>
                </thead>
                <tbody className='[&_tr:last-child]:border-0'>
                  {items.map((item) => {
                    const statusKey = sessionStatusLabelKey(item.status)
                    return (
                      <tr
                        key={item.id}
                        className='group/row border-b transition-colors hover:bg-muted/50'
                      >
                        <td className='p-2 align-middle font-medium'>
                          {item.id}
                        </td>
                        <td className='p-2 align-middle'>
                          {statusKey ? t(statusKey) : item.status}
                        </td>
                        <td className='text-muted-foreground p-2 align-middle'>
                          {item.user_id}
                        </td>
                        <td className='text-muted-foreground p-2 align-middle'>
                          {item.case_id}
                        </td>
                        <td className='text-muted-foreground p-2 align-middle'>
                          {item.chat_id}
                        </td>
                        <td className='text-muted-foreground p-2 align-middle'>
                          {item.created_at}
                        </td>
                        <td className='p-2 align-middle'>
                          <Button
                            type='button'
                            variant='outline'
                            size='sm'
                            onClick={() => onOpenDetail(item)}
                          >
                            {t('common.detail')}
                          </Button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>

          <div className='flex flex-col gap-4 pt-3 md:hidden'>
            {items.map((item) => {
              const statusKey = sessionStatusLabelKey(item.status)
              return (
                <article
                  key={item.id}
                  className='border-border/70 bg-card rounded-2xl border p-4'
                >
                  <div className='flex items-start justify-between gap-3'>
                    <div className='min-w-0'>
                      <p className='truncate text-base font-medium'>
                        {item.id}
                      </p>
                      <p className='text-muted-foreground mt-1 text-sm'>
                        {statusKey ? t(statusKey) : item.status}
                      </p>
                    </div>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      className='shrink-0'
                      onClick={() => onOpenDetail(item)}
                    >
                      {t('common.detail')}
                    </Button>
                  </div>
                  <div className='mt-3 grid grid-cols-2 gap-3 text-sm'>
                    <div className='flex flex-col gap-1'>
                      <span className='text-muted-foreground text-xs'>
                        {t('sessions.fieldUserId')}
                      </span>
                      <span>{item.user_id}</span>
                    </div>
                    <div className='flex flex-col gap-1'>
                      <span className='text-muted-foreground text-xs'>
                        {t('sessions.fieldCaseId')}
                      </span>
                      <span>{item.case_id}</span>
                    </div>
                  </div>
                </article>
              )
            })}
          </div>
        </>
      ) : null}
    </div>
  )
}

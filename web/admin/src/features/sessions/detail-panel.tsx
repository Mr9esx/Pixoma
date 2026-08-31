import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Reveal } from '@/components/ui/reveal'
import { getSession } from '@/lib/api/sessions'
import { queryKeys } from '@/lib/api/query-keys'
import type { SessionRecord } from '@/lib/api/types'
import { sessionStatusLabelKey } from './list-panel'

type Props = {
  id: string
}

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function Field({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className='grid gap-1 sm:grid-cols-[10rem_1fr] sm:items-start'>
      <dt className='text-muted-foreground text-xs font-medium'>{label}</dt>
      <dd className='text-sm break-all'>{value || '—'}</dd>
    </div>
  )
}

function JsonBlock({ value }: { value: unknown }) {
  return (
    <pre className='bg-muted max-h-96 overflow-auto rounded-md p-3 text-xs whitespace-pre-wrap break-all'>
      {JSON.stringify(value, null, 2)}
    </pre>
  )
}

function SessionFields({
  session,
  t,
}: {
  session: SessionRecord
  t: (k: string) => string
}) {
  const statusKey = sessionStatusLabelKey(session.status)
  return (
    <dl className='space-y-3'>
      <Field label={t('sessions.fieldId')} value={session.id} />
      <Field
        label={t('sessions.fieldStatus')}
        value={statusKey ? t(statusKey) : session.status}
      />
      <Field label={t('sessions.fieldUserId')} value={session.user_id} />
      <Field
        label={t('sessions.fieldPlatform')}
        value={session.channel_name || session.channel_id}
      />
      <Field label={t('sessions.fieldChatId')} value={String(session.chat_id)} />
      <Field label={t('sessions.fieldCaseId')} value={session.case_id} />
      <Field
        label={t('sessions.fieldCurrentInputIndex')}
        value={String(session.current_input_index)}
      />
      <Field
        label={t('sessions.fieldInputKeys')}
        value={
          session.input_keys.length > 0
            ? session.input_keys.join(', ')
            : undefined
        }
      />
      <Field label={t('sessions.fieldCreatedAt')} value={session.created_at} />
      <Field label={t('sessions.fieldUpdatedAt')} value={session.updated_at} />
      <div className='space-y-2'>
        <dt className='text-muted-foreground text-xs font-medium'>
          {t('sessions.fieldDraft')}
        </dt>
        <dd>
          <JsonBlock value={session.draft} />
        </dd>
      </div>
    </dl>
  )
}

export function SessionDetailPanel({ id }: Props) {
  const { t } = useTranslation()

  const detailQuery = useQuery({
    queryKey: queryKeys.sessions.detail(id),
    queryFn: () => getSession(id),
  })

  if (detailQuery.isLoading) {
    return (
      <div data-testid='session-detail-panel'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div data-testid='session-detail-panel' className='space-y-3'>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }

  const session = detailQuery.data
  if (!session) return null

  return (
    <Reveal className='space-y-4' data-testid='session-detail-panel'>
      <div>
        <h2 className='text-lg font-semibold'>{session.id}</h2>
        <p className='text-muted-foreground text-sm'>
          {t('sessions.detailHeading')}
        </p>
      </div>
      <SessionFields session={session} t={t} />
    </Reveal>
  )
}

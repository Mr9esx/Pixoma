import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { getSession } from '@/lib/api/sessions'
import type { SessionDraft } from '@/lib/api/types'
import { formatDateTime, formatUserLabel } from '@/lib/format'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import {
  RelatedChip,
  ResourceDetailFallbackTitle,
  ResourceDetailLayout,
  type IdentifierItem,
} from '@/features/operations/detail-layout'
import { formatDraftView } from '@/features/operations/draft-value'
import {
  lifecycleBadgeClass,
  opsIdentity,
} from '@/features/operations/identity'
import type { OperationsDetailTarget } from '@/features/operations/types'
import { resourceDetailBodyClassName } from '@/features/resource-modal'
import { sessionStatusLabelKey } from './list-panel'

type Props = {
  id: string
  onOpenRelated?: (target: OperationsDetailTarget) => void
}

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function DraftList({
  draft,
  title,
  skippedLabel,
}: {
  draft: Record<string, SessionDraft>
  title: string
  skippedLabel: string
}) {
  const entries = Object.values(draft)
  if (entries.length === 0) return null
  return (
    <section className='flex flex-col gap-2'>
      <h3 className='text-xs font-medium text-muted-foreground'>{title}</h3>
      <dl className='flex flex-col gap-3'>
        {entries.map((entry) => {
          const view = formatDraftView(entry)
          return (
            <div key={entry.key} className='flex min-w-0 flex-col gap-1'>
              <dt className='font-mono text-xs font-medium text-muted-foreground'>
                {entry.key}
              </dt>
              <dd className='text-sm break-all'>
                {view.kind === 'skipped' ? skippedLabel : view.text || '—'}
              </dd>
            </div>
          )
        })}
      </dl>
    </section>
  )
}

export function SessionDetailPanel({ id, onOpenRelated }: Props) {
  const { t } = useTranslation()

  const detailQuery = useQuery({
    queryKey: queryKeys.sessions.detail(id),
    queryFn: () => getSession(id),
  })

  if (detailQuery.isLoading) {
    return (
      <>
        <ResourceDetailFallbackTitle>
          {t('sessions.detailHeading')}
        </ResourceDetailFallbackTitle>
        <div
          data-testid='session-detail-panel'
          className={resourceDetailBodyClassName}
        >
          <LoadingSkeleton rows={8} />
        </div>
      </>
    )
  }

  if (detailQuery.isError) {
    return (
      <>
        <ResourceDetailFallbackTitle>
          {t('sessions.detailHeading')}
        </ResourceDetailFallbackTitle>
        <div
          data-testid='session-detail-panel'
          className={`${resourceDetailBodyClassName} flex flex-col gap-3`}
        >
          <ErrorBanner
            message={errorMessage(detailQuery.error)}
            onRetry={() => void detailQuery.refetch()}
          />
        </div>
      </>
    )
  }

  const session = detailQuery.data
  if (!session) return null

  const userLabel = formatUserLabel(session.user, session.user_id)
  const title =
    opsIdentity({ userLabel, caseId: session.case_id }) || session.id
  const statusKey = sessionStatusLabelKey(session.status)
  const statusLabel = statusKey ? t(statusKey) : session.status
  const progress = `${session.current_input_index} / ${session.input_keys.length}`

  const identifiers: IdentifierItem[] = [
    {
      label: t('sessions.fieldChatId'),
      value: String(session.chat_id),
      copy: true,
    },
    { label: t('sessions.fieldProgress'), value: progress },
    {
      label: t('sessions.fieldCreatedAt'),
      value: formatDateTime(session.created_at),
    },
    {
      label: t('sessions.fieldUpdatedAt'),
      value: formatDateTime(session.updated_at),
    },
  ]

  return (
    <ResourceDetailLayout
      testId='session-detail-panel'
      title={title}
      status={statusLabel}
      statusClassName={lifecycleBadgeClass(session.status)}
      recordId={session.id}
      related={
        <>
          <RelatedChip
            label={t('sessions.fieldUser')}
            value={userLabel || session.user_id}
            onClick={
              onOpenRelated
                ? () => onOpenRelated({ kind: 'user', id: session.user_id })
                : undefined
            }
          />
          {session.channel_id ? (
            <RelatedChip
              label={t('sessions.fieldPlatform')}
              value={session.channel_name || session.channel_id}
              to='/channels/$id'
              params={{ id: session.channel_id }}
            />
          ) : session.channel_name ? (
            <RelatedChip
              label={t('sessions.fieldPlatform')}
              value={session.channel_name}
            />
          ) : null}
          <RelatedChip
            label={t('sessions.fieldCaseId')}
            value={String(session.case_id)}
            to='/cases/$caseId'
            params={{ caseId: String(session.case_id) }}
          />
        </>
      }
      identifiers={identifiers}
      extra={
        <DraftList
          draft={session.draft}
          title={t('sessions.fieldDraft')}
          skippedLabel={t('sessions.draftSkipped')}
        />
      }
    />
  )
}

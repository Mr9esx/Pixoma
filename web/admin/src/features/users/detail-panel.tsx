import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { getUser } from '@/lib/api/users'
import { formatDateTime, formatUserLabel } from '@/lib/format'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import {
  RelatedChip,
  ResourceDetailFallbackTitle,
  ResourceDetailLayout,
  type IdentifierItem,
} from '@/features/operations/detail-layout'
import { lifecycleBadgeClass } from '@/features/operations/identity'
import { resourceDetailBodyClassName } from '@/features/resource-modal'

type Props = {
  id: string
}

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function UserDetailPanel({ id }: Props) {
  const { t } = useTranslation()

  const detailQuery = useQuery({
    queryKey: queryKeys.users.detail(id),
    queryFn: () => getUser(id),
  })

  if (detailQuery.isLoading) {
    return (
      <>
        <ResourceDetailFallbackTitle>
          {t('users.detailHeading')}
        </ResourceDetailFallbackTitle>
        <div
          data-testid='user-detail-panel'
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
          {t('users.detailHeading')}
        </ResourceDetailFallbackTitle>
        <div
          data-testid='user-detail-panel'
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

  const user = detailQuery.data
  if (!user) return null

  const title = formatUserLabel(user, user.external_user_id) || user.id
  const accessLabel =
    user.access === 'always_allowed'
      ? t('users.accessAlwaysAllowed')
      : user.access === 'paid'
        ? t('users.accessPaid')
        : t('users.accessDenied')
  const accessClass =
    user.access === 'denied'
      ? 'border-destructive/25 bg-destructive/10 text-destructive'
      : lifecycleBadgeClass('succeeded')

  const identifiers: IdentifierItem[] = [
    {
      label: t('users.fieldExternalUserId'),
      value: user.external_user_id,
      copy: true,
    },
    user.username && user.username !== title
      ? { label: t('users.fieldUsername'), value: user.username }
      : null,
    user.language_code
      ? { label: t('users.fieldLanguageCode'), value: user.language_code }
      : null,
    {
      label: t('users.fieldLastSeenAt'),
      value: formatDateTime(user.last_seen_at),
    },
    {
      label: t('users.fieldCreatedAt'),
      value: formatDateTime(user.created_at),
    },
    {
      label: t('users.fieldUpdatedAt'),
      value: formatDateTime(user.updated_at),
    },
  ].filter((item): item is IdentifierItem => item != null)

  return (
    <ResourceDetailLayout
      testId='user-detail-panel'
      title={title}
      status={accessLabel}
      statusClassName={accessClass}
      recordId={user.id}
      related={
        user.channel_id ? (
          <RelatedChip
            label={t('users.fieldPlatform')}
            value={user.channel_name || user.channel_id}
            to='/channels/$id'
            params={{ id: user.channel_id }}
          />
        ) : user.channel_name ? (
          <RelatedChip
            label={t('users.fieldPlatform')}
            value={user.channel_name}
          />
        ) : null
      }
      identifiers={identifiers}
    />
  )
}

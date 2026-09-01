import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import type { UserRecord } from '@/lib/api/types'
import { getUser } from '@/lib/api/users'
import { formatDateTime } from '@/lib/format'
import { Reveal } from '@/components/ui/reveal'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { DetailField as Field } from '@/components/detail-field'

type Props = {
  id: string
}

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function UserFields({
  user,
  t,
}: {
  user: UserRecord
  t: (k: string) => string
}) {
  const accessLabel =
    user.access === 'always_allowed'
      ? t('users.accessAlwaysAllowed')
      : user.access === 'paid'
        ? t('users.accessPaid')
        : t('users.accessDenied')

  return (
    <dl className='space-y-3'>
      <Field label={t('users.fieldId')} value={user.id} />
      <Field label={t('users.fieldPlatform')} value={user.channel_name} />
      <Field
        label={t('users.fieldExternalUserId')}
        value={user.external_user_id}
      />
      <Field label={t('users.fieldUsername')} value={user.username} />
      <Field label={t('users.fieldFirstName')} value={user.first_name} />
      <Field label={t('users.fieldLastName')} value={user.last_name} />
      <Field label={t('users.fieldLanguageCode')} value={user.language_code} />
      <Field label={t('users.fieldAccess')} value={accessLabel} />
      <Field
        label={t('users.fieldLastSeenAt')}
        value={formatDateTime(user.last_seen_at)}
      />
      <Field
        label={t('users.fieldCreatedAt')}
        value={formatDateTime(user.created_at)}
      />
      <Field
        label={t('users.fieldUpdatedAt')}
        value={formatDateTime(user.updated_at)}
      />
    </dl>
  )
}

export function UserDetailPanel({ id }: Props) {
  const { t } = useTranslation()

  const detailQuery = useQuery({
    queryKey: queryKeys.users.detail(id),
    queryFn: () => getUser(id),
  })

  if (detailQuery.isLoading) {
    return (
      <div data-testid='user-detail-panel'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div data-testid='user-detail-panel' className='space-y-3'>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }

  const user = detailQuery.data
  if (!user) return null

  return (
    <Reveal className='space-y-4' data-testid='user-detail-panel'>
      <div>
        <h2 className='text-lg font-semibold'>{user.id}</h2>
        <p className='text-sm text-muted-foreground'>
          {t('users.detailHeading')}
        </p>
      </div>
      <UserFields user={user} t={t} />
    </Reveal>
  )
}

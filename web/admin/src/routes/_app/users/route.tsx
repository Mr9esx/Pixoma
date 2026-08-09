import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { listUsers } from '@/lib/api/users'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { UserDetailPanel } from '@/features/users/detail-panel'
import {
  UserListPanel,
  type UserListFilters,
} from '@/features/users/list-panel'

export const Route = createFileRoute('/_app/users')({
  component: UsersLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function UsersLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { userId } = useParams({ strict: false }) as { userId?: string }

  const [filters, setFilters] = useState<UserListFilters>({
    q: '',
  })

  const listParams = {
    q: filters.q.trim() || undefined,
  }

  const listQuery = useQuery({
    queryKey: [...queryKeys.users.all, listParams] as const,
    queryFn: () => listUsers(listParams),
  })

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='users-page'
    >
      <div className='shrink-0'>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('users.title')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('users.description')}
        </p>
      </div>
      <MasterDetailShell
        hasSelection={Boolean(userId)}
        onBackToList={() => void navigate({ to: '/users' })}
        list={
          <UserListPanel
            items={listQuery.data ?? []}
            selectedId={userId}
            filters={filters}
            onFiltersChange={setFilters}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={userId ? <UserDetailPanel id={userId} /> : null}
      />
    </div>
  )
}

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  useNavigate,
  useParams,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { UserDetailPanel } from '@/features/users/detail-panel'
import {
  UserListPanel,
  type UserListFilters,
} from '@/features/users/list-panel'
import { listUsers } from '@/lib/api/users'
import { queryKeys } from '@/lib/api/query-keys'

export const Route = createFileRoute('/_app/users')({
  component: UsersLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function parseTgUserId(raw: string): number | undefined {
  const trimmed = raw.trim()
  if (!trimmed) return undefined
  const n = Number(trimmed)
  return Number.isFinite(n) ? n : undefined
}

function UsersLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { userId } = useParams({ strict: false }) as { userId?: string }

  const [filters, setFilters] = useState<UserListFilters>({
    q: '',
    tg_user_id: '',
  })

  const listParams = {
    q: filters.q.trim() || undefined,
    tg_user_id: parseTgUserId(filters.tg_user_id),
  }

  const listQuery = useQuery({
    queryKey: [...queryKeys.users.all, listParams] as const,
    queryFn: () => listUsers(listParams),
  })

  return (
    <div className='space-y-3' data-testid='users-page'>
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('users.title')}
        </h1>
        <p className='text-muted-foreground text-sm'>
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

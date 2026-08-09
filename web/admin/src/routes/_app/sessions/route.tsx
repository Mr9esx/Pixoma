import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  useNavigate,
  useParams,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { SessionDetailPanel } from '@/features/sessions/detail-panel'
import {
  SessionListPanel,
  type SessionListFilters,
} from '@/features/sessions/list-panel'
import { listSessions } from '@/lib/api/sessions'
import { queryKeys } from '@/lib/api/query-keys'

export const Route = createFileRoute('/_app/sessions')({
  component: SessionsLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function SessionsLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { sessionId } = useParams({ strict: false }) as {
    sessionId?: string
  }

  const [filters, setFilters] = useState<SessionListFilters>({
    user_id: '',
    status: 'all',
    q: '',
  })

  const listParams = {
    user_id: filters.user_id.trim() || undefined,
    status: filters.status === 'all' ? undefined : filters.status,
    q: filters.q.trim() || undefined,
  }

  const listQuery = useQuery({
    queryKey: [...queryKeys.sessions.all, listParams] as const,
    queryFn: () => listSessions(listParams),
  })

  return (
    <div className='space-y-3' data-testid='sessions-page'>
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('sessions.title')}
        </h1>
        <p className='text-muted-foreground text-sm'>
          {t('sessions.description')}
        </p>
      </div>
      <MasterDetailShell
        hasSelection={Boolean(sessionId)}
        onBackToList={() => void navigate({ to: '/sessions' })}
        list={
          <SessionListPanel
            items={listQuery.data ?? []}
            selectedId={sessionId}
            filters={filters}
            onFiltersChange={setFilters}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={sessionId ? <SessionDetailPanel id={sessionId} /> : null}
      />
    </div>
  )
}

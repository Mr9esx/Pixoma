import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { listSessions } from '@/lib/api/sessions'
import { OperationsDetailDialog } from '@/features/operations/detail-dialog'
import type { OperationsDetailTarget } from '@/features/operations/types'
import { SessionListPanel } from '@/features/sessions/list-panel'

export const Route = createFileRoute('/_app/sessions')({
  component: SessionsLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function SessionsLayout() {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<OperationsDetailTarget | null>(null)

  const listQuery = useQuery({
    queryKey: queryKeys.sessions.all,
    queryFn: () => listSessions({}),
  })
  const items = listQuery.data ?? []

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='sessions-page'
    >
      <div className='flex shrink-0 flex-col gap-1'>
        <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
          {t('sessions.title')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('sessions.description')}
        </p>
      </div>

      <SessionListPanel
        items={items}
        onOpenDetail={setDetail}
        isLoading={listQuery.isLoading}
        isError={listQuery.isError}
        errorMessage={errorMessage(listQuery.error)}
        onRetry={() => void listQuery.refetch()}
      />

      <OperationsDetailDialog
        target={detail}
        onClose={() => setDetail(null)}
        onOpenRelated={setDetail}
      />
    </div>
  )
}

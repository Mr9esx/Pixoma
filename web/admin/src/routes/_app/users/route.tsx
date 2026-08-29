import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import type { UserRecord } from '@/lib/api/types'
import { listUsers } from '@/lib/api/users'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { UserDetailPanel } from '@/features/users/detail-panel'
import { UserListPanel } from '@/features/users/list-panel'

export const Route = createFileRoute('/_app/users')({
  component: UsersLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function UsersLayout() {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<UserRecord | null>(null)

  const listQuery = useQuery({
    queryKey: queryKeys.users.all,
    queryFn: () => listUsers({}),
  })
  const items = listQuery.data ?? []

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='users-page'
    >
      <div className='flex shrink-0 flex-col gap-1'>
        <h1 className='truncate text-2xl leading-tight font-semibold tracking-tight'>
          {t('users.title')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('users.description')}
        </p>
      </div>

      <UserListPanel
        items={items}
        onOpenDetail={setDetail}
        isLoading={listQuery.isLoading}
        isError={listQuery.isError}
        errorMessage={errorMessage(listQuery.error)}
        onRetry={() => void listQuery.refetch()}
      />

      <Dialog
        open={detail !== null}
        onOpenChange={(open) => {
          if (!open) setDetail(null)
        }}
      >
        <DialogContent className='flex max-h-[85vh] flex-col sm:max-w-3xl'>
          <DialogHeader>
            <DialogTitle>{t('users.detailHeading')}</DialogTitle>
          </DialogHeader>
          <div className='min-h-0 flex-1 overflow-auto px-5 py-4'>
            {detail ? <UserDetailPanel id={detail.id} /> : null}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

import { useMemo, useState } from 'react'
import {
  createColumnHelper,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Eye } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { UserRecord } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { DataTableColumnHeader } from '@/components/data-table/column-header'
import { DataTable } from '@/components/data-table/data-table'
import { DataTableToolbar } from '@/components/data-table/toolbar'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Reveal } from '@/components/ui/reveal'

type Props = {
  items: UserRecord[]
  onOpenDetail: (item: UserRecord) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function UserListPanel({
  items,
  onOpenDetail,
  isLoading,
  isError,
  errorMessage,
  onRetry,
}: Props) {
  const { t } = useTranslation()

  const columnHelper = useMemo(() => createColumnHelper<UserRecord>(), [])

  const columns = useMemo(
    () => [
      columnHelper.accessor('id', {
        id: 'id',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('users.fieldId')} />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs'>{getValue()}</span>
        ),
        enableHiding: false,
      }),
      columnHelper.accessor('tg_user_id', {
        id: 'tg_user_id',
        meta: { label: t('users.fieldTgUserId') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('users.fieldTgUserId')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs tabular-nums'>{getValue()}</span>
        ),
      }),
      columnHelper.accessor('username', {
        id: 'username',
        meta: { label: t('users.fieldUsername') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('users.fieldUsername')}
          />
        ),
        cell: ({ getValue }) => getValue() || '—',
      }),
      columnHelper.accessor('created_at', {
        id: 'created_at',
        meta: { label: t('users.fieldCreatedAt') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('users.fieldCreatedAt')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='whitespace-nowrap text-muted-foreground'>
            {getValue()}
          </span>
        ),
      }),
      columnHelper.accessor('last_seen_at', {
        id: 'last_seen_at',
        meta: { label: t('users.fieldLastSeenAt') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('users.fieldLastSeenAt')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='whitespace-nowrap text-muted-foreground'>
            {getValue() || '—'}
          </span>
        ),
      }),
      columnHelper.display({
        id: 'actions',
        header: () => <span className='sr-only'>{t('common.detail')}</span>,
        cell: ({ row }) => (
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => onOpenDetail(row.original)}
          >
            <Eye className='size-4' />
            {t('common.detail')}
          </Button>
        ),
        enableHiding: false,
      }),
    ],
    [columnHelper, t, onOpenDetail]
  )

  const [sorting, setSorting] = useState([{ id: 'created_at', desc: true }])
  const [columnVisibility, setColumnVisibility] = useState({})
  const [globalFilter, setGlobalFilter] = useState('')

  const table = useReactTable({
    data: items,
    columns,
    state: { sorting, columnVisibility, globalFilter },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn: (row, _columnId, filterValue) => {
      const q = String(filterValue ?? '')
        .trim()
        .toLowerCase()
      if (!q) return true
      const { id, tg_user_id, username } = row.original
      return [id, String(tg_user_id), username].some((value) =>
        String(value).toLowerCase().includes(q)
      )
    },
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: { pagination: { pageIndex: 0, pageSize: 10 } },
  })

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='users-list-panel'
    >
      {isError ? (
        <ErrorBanner message={errorMessage} onRetry={onRetry} />
      ) : null}

      {isLoading ? <LoadingSkeleton rows={5} /> : null}

      {!isLoading && !isError ? (
        <Reveal>
          <DataTable
            table={table}
            emptyState={
              <EmptyState className='py-8' message={t('users.empty')} />
            }
          >
            <DataTableToolbar
              table={table}
              searchPlaceholder={t('users.filterQPlaceholder')}
            />
          </DataTable>
        </Reveal>
      ) : null}
    </div>
  )
}

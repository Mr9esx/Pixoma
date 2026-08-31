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
import { Check, Eye, MoreHorizontal } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { UserRecord } from '@/lib/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Reveal } from '@/components/ui/reveal'
import { DataTableColumnHeader } from '@/components/data-table/column-header'
import { DataTable } from '@/components/data-table/data-table'
import { DataTableToolbar } from '@/components/data-table/toolbar'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'

type Props = {
  items: UserRecord[]
  onOpenDetail: (item: UserRecord) => void
  onSetAccess: (item: UserRecord, access: UserRecord['access']) => void
  setAccessPendingId?: string
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function UserListPanel({
  items,
  onOpenDetail,
  onSetAccess,
  setAccessPendingId,
  isLoading,
  isError,
  errorMessage,
  onRetry,
}: Props) {
  const { t } = useTranslation()

  const columnHelper = useMemo(() => createColumnHelper<UserRecord>(), [])

  const accessOptions = useMemo(
    () =>
      [
        { value: 'always_allowed', label: t('users.accessAlwaysAllowed') },
        { value: 'paid', label: t('users.accessPaid') },
        { value: 'denied', label: t('users.accessDenied') },
      ] as Array<{ value: UserRecord['access']; label: string }>,
    [t]
  )

  const accessLabel = (access: UserRecord['access']) =>
    accessOptions.find((option) => option.value === access)?.label ??
    t('users.accessDenied')

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
      columnHelper.accessor('access', {
        id: 'access',
        meta: { label: t('users.fieldAccess') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('users.fieldAccess')}
          />
        ),
        cell: ({ getValue }) => {
          const access = getValue()
          return (
            <Badge
              variant={
                access === 'always_allowed'
                  ? 'default'
                  : access === 'paid'
                    ? 'secondary'
                    : 'destructive'
              }
            >
              {accessLabel(access)}
            </Badge>
          )
        },
      }),
      columnHelper.display({
        id: 'actions',
        header: () => <span className='sr-only'>{t('common.actions')}</span>,
        cell: ({ row }) => (
          <div className='flex items-center justify-end gap-1'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => onOpenDetail(row.original)}
            >
              <Eye className='size-4' />
              {t('common.detail')}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type='button'
                  variant='ghost'
                  size='sm'
                  className='size-8 p-0'
                  disabled={setAccessPendingId === row.original.id}
                  aria-label={`${row.original.username || row.original.id} ${t('users.access')}`}
                >
                  <MoreHorizontal className='size-4' />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align='end'>
                {accessOptions.map((option) => (
                  <DropdownMenuItem
                    key={option.value}
                    disabled={row.original.access === option.value}
                    onSelect={() => onSetAccess(row.original, option.value)}
                  >
                    <Check
                      className={
                        row.original.access === option.value
                          ? 'opacity-100'
                          : 'opacity-0'
                      }
                    />
                    {option.label}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        ),
        enableHiding: false,
      }),
    ],
    [
      columnHelper,
      t,
      onOpenDetail,
      onSetAccess,
      accessOptions,
      setAccessPendingId,
    ]
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

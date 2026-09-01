import { useMemo, useState } from 'react'
import { Link } from '@tanstack/react-router'
import {
  createColumnHelper,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type ColumnFiltersState,
  useReactTable,
} from '@tanstack/react-table'
import type { Option } from '@/types/data-table'
import { Eye } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SessionRecord } from '@/lib/api/types'
import { formatDateTime, formatUserLabel } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { FadeSwap } from '@/components/ui/fade-swap'
import { Reveal } from '@/components/ui/reveal'
import { DataTableColumnHeader } from '@/components/data-table/column-header'
import { DataTable } from '@/components/data-table/data-table'
import { DataTableToolbar } from '@/components/data-table/toolbar'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'

export function sessionStatusLabelKey(status: string): string | undefined {
  const map: Record<string, string> = {
    collecting: 'sessions.statusCollecting',
    confirming: 'sessions.statusConfirming',
    submitted: 'sessions.statusSubmitted',
    exited: 'sessions.statusExited',
  }
  return map[status]
}

type Props = {
  items: SessionRecord[]
  onOpenDetail: (item: SessionRecord) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function SessionListPanel({
  items,
  onOpenDetail,
  isLoading,
  isError,
  errorMessage,
  onRetry,
}: Props) {
  const { t } = useTranslation()

  const statusOptions = useMemo<Option[]>(
    () =>
      ['collecting', 'confirming', 'submitted', 'exited'].map((value) => {
        const key = sessionStatusLabelKey(value)
        return { value, label: key ? t(key) : value }
      }),
    [t]
  )

  const columnHelper = useMemo(() => createColumnHelper<SessionRecord>(), [])

  const columns = useMemo(() => {
    const renderStatus = (status: string) => {
      const key = sessionStatusLabelKey(status)
      return key ? t(key) : status
    }

    return [
      columnHelper.accessor('id', {
        id: 'id',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('sessions.fieldId')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs'>{getValue()}</span>
        ),
        enableHiding: false,
      }),
      columnHelper.accessor('status', {
        id: 'status',
        meta: { label: t('sessions.fieldStatus') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('sessions.fieldStatus')}
          />
        ),
        cell: ({ getValue }) => {
          const label = renderStatus(getValue())
          return (
            <FadeSwap value={label} className='text-muted-foreground'>
              {label}
            </FadeSwap>
          )
        },
        filterFn: (row, id, value) => {
          const selected = value as string[] | undefined
          return (
            !selected?.length || selected.includes(row.getValue(id) as string)
          )
        },
      }),
      columnHelper.accessor('user_id', {
        id: 'user_id',
        meta: { label: t('sessions.fieldUser') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('sessions.fieldUser')}
          />
        ),
        cell: ({ row }) => {
          const label = formatUserLabel(row.original.user, row.original.user_id)
          if (!label) return '—'
          return (
            <Link
              to='/users/$userId'
              params={{ userId: row.original.user_id }}
              className='text-foreground underline-offset-4 hover:underline'
            >
              {label}
            </Link>
          )
        },
      }),
      columnHelper.accessor('channel_name', {
        id: 'channel_id',
        meta: { label: t('sessions.fieldPlatform') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('sessions.fieldPlatform')}
          />
        ),
        cell: ({ getValue }) => getValue() || '—',
      }),
      columnHelper.accessor('case_id', {
        id: 'case_id',
        meta: { label: t('sessions.fieldCaseId') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('sessions.fieldCaseId')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs tabular-nums'>{getValue()}</span>
        ),
      }),
      columnHelper.accessor('chat_id', {
        id: 'chat_id',
        meta: { label: t('sessions.fieldChatId') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('sessions.fieldChatId')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs tabular-nums'>{getValue()}</span>
        ),
      }),
      columnHelper.accessor('created_at', {
        id: 'created_at',
        meta: { label: t('sessions.fieldCreatedAt') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('sessions.fieldCreatedAt')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='whitespace-nowrap text-muted-foreground'>
            {formatDateTime(getValue())}
          </span>
        ),
      }),
      columnHelper.display({
        id: 'actions',
        header: () => <span>{t('common.actions')}</span>,
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
    ]
  }, [columnHelper, t, onOpenDetail])

  const [sorting, setSorting] = useState([{ id: 'created_at', desc: true }])
  const [columnVisibility, setColumnVisibility] = useState({})
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [globalFilter, setGlobalFilter] = useState('')

  const table = useReactTable({
    data: items,
    columns,
    state: { sorting, columnVisibility, columnFilters, globalFilter },
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    globalFilterFn: (row, _columnId, filterValue) => {
      const q = String(filterValue ?? '')
        .trim()
        .toLowerCase()
      if (!q) return true
      const { id, user_id, case_id, chat_id } = row.original
      return [
        id,
        user_id,
        case_id,
        chat_id,
        row.original.channel_name,
        row.original.channel_id,
      ].some((value) =>
        String(value ?? '')
          .toLowerCase()
          .includes(q)
      )
    },
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: {
      pagination: { pageIndex: 0, pageSize: 10 },
      columnPinning: { right: ['actions'] },
    },
  })

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='sessions-list-panel'
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
              <EmptyState className='py-8' message={t('sessions.empty')} />
            }
          >
            <DataTableToolbar
              table={table}
              searchPlaceholder={t('sessions.filterQPlaceholder')}
              filters={[
                {
                  columnId: 'status',
                  title: t('sessions.fieldStatus'),
                  options: statusOptions,
                },
              ]}
            />
          </DataTable>
        </Reveal>
      ) : null}
    </div>
  )
}

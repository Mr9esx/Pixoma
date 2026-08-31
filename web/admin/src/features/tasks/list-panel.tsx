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
import type { TaskRecord } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { DataTableColumnHeader } from '@/components/data-table/column-header'
import { DataTable } from '@/components/data-table/data-table'
import { DataTableToolbar } from '@/components/data-table/toolbar'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { FadeSwap } from '@/components/ui/fade-swap'
import { Reveal } from '@/components/ui/reveal'

export function taskStatusLabelKey(status: string): string | undefined {
  const map: Record<string, string> = {
    pending: 'tasks.statusPending',
    queued: 'tasks.statusQueued',
    running: 'tasks.statusRunning',
    succeeded: 'tasks.statusSucceeded',
    failed: 'tasks.statusFailed',
    cancelled: 'tasks.statusCancelled',
  }
  return map[status]
}

type Props = {
  items: TaskRecord[]
  onOpenDetail: (item: TaskRecord) => void
  isLoading?: boolean
  isError?: boolean
  errorMessage?: string
  onRetry?: () => void
}

export function TaskListPanel({
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
      ['pending', 'queued', 'running', 'succeeded', 'failed', 'cancelled'].map(
        (value) => {
          const key = taskStatusLabelKey(value)
          return {
            value,
            label: key ? t(key) : value,
          }
        }
      ),
    [t]
  )

  const columnHelper = useMemo(() => createColumnHelper<TaskRecord>(), [])

  const columns = useMemo(() => {
    const renderStatus = (status: string) => {
      const key = taskStatusLabelKey(status)
      return key ? t(key) : status
    }

    return [
      columnHelper.accessor('id', {
        id: 'id',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('tasks.fieldId')} />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs tabular-nums'>{getValue()}</span>
        ),
        enableHiding: false,
      }),
      columnHelper.accessor('status', {
        id: 'status',
        meta: { label: t('tasks.fieldStatus') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('tasks.fieldStatus')}
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
      columnHelper.accessor('case_id', {
        id: 'case_id',
        meta: { label: t('tasks.fieldCaseId') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('tasks.fieldCaseId')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs tabular-nums'>{getValue()}</span>
        ),
      }),
      columnHelper.accessor((row) => row.channel_name || row.channel_id, {
        id: 'channel_id',
        meta: { label: t('tasks.fieldPlatform') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('tasks.fieldPlatform')}
          />
        ),
        cell: ({ getValue }) => getValue() || '—',
      }),
      columnHelper.accessor('user_id', {
        id: 'user_id',
        meta: { label: t('tasks.fieldUser') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('tasks.fieldUser')}
          />
        ),
        cell: ({ row }) => {
          const user = row.original.user
          const label =
            user?.username ||
            [user?.first_name, user?.last_name].filter(Boolean).join(' ') ||
            row.original.user_id
          if (!label) return '—'
          return (
            <Link
              to='/users/$userId'
              params={{ userId: row.original.user_id! }}
              className='text-foreground underline-offset-4 hover:underline'
            >
              {label}
            </Link>
          )
        },
      }),
      columnHelper.accessor('session_id', {
        id: 'session_id',
        meta: { label: t('tasks.fieldSessionId') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('tasks.fieldSessionId')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='font-mono text-xs'>
            {getValue() ? (
              <Link
                to='/sessions/$sessionId'
                params={{ sessionId: getValue()! }}
                className='text-foreground underline-offset-4 hover:underline'
              >
                {getValue()}
              </Link>
            ) : (
              '—'
            )}
          </span>
        ),
      }),
      columnHelper.accessor('edge_id', {
        id: 'edge_id',
        meta: { label: t('tasks.fieldEdgeId') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('tasks.fieldEdgeId')}
          />
        ),
        cell: ({ getValue }) => getValue() ?? '—',
      }),
      columnHelper.accessor('created_at', {
        id: 'created_at',
        meta: { label: t('tasks.fieldCreatedAt') },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('tasks.fieldCreatedAt')}
          />
        ),
        cell: ({ getValue }) => (
          <span className='whitespace-nowrap text-muted-foreground'>
            {getValue()}
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
      const { id, case_id, session_id, edge_id } = row.original
      return [
        id,
        case_id,
        session_id,
        edge_id,
        row.original.channel_name,
        row.original.channel_id,
        row.original.user?.username,
        row.original.user_id,
      ].some((value) => String(value ?? '').toLowerCase().includes(q))
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
      data-testid='tasks-list-panel'
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
              <EmptyState className='py-8' message={t('tasks.empty')} />
            }
          >
            <DataTableToolbar
              table={table}
              searchPlaceholder={t('tasks.filterQPlaceholder')}
              filters={[
                {
                  columnId: 'status',
                  title: t('tasks.fieldStatus'),
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

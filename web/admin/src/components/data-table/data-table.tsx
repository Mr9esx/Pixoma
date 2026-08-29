import { flexRender, type Table as TanstackTable } from '@tanstack/react-table'
import { getColumnPinningStyle } from '@/lib/data-table'
import { cn } from '@/lib/utils'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { DataTablePagination } from './pagination'

type DataTableProps<TData> = React.ComponentProps<'div'> & {
  table: TanstackTable<TData>
  actionBar?: React.ReactNode
  emptyState?: React.ReactNode
  /** 行点击回调（用于 master-detail 联动的表格）。 */
  onRowClick?: (original: TData) => void
  /** 高亮选中的行（可选）。 */
  selectedRowId?: string
  /** 行 id 提取函数，配合 selectedRowId 使用。 */
  getRowId?: (original: TData) => string
  /** 行数较少时隐藏分页器（默认关闭）。 */
  hidePagination?: boolean
}

export function DataTable<TData>({
  table,
  actionBar,
  emptyState,
  onRowClick,
  selectedRowId,
  getRowId,
  hidePagination = false,
  className,
  children,
  ...props
}: DataTableProps<TData>) {
  const rows = table.getRowModel().rows

  return (
    <div
      className={cn('flex w-full flex-col gap-3 overflow-auto', className)}
      {...props}
    >
      {children ? <div className='px-1 first:pt-1'>{children}</div> : null}
      <div className='overflow-hidden rounded-lg border bg-background'>
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className='bg-muted/40'>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    colSpan={header.colSpan}
                    className='p-0'
                    style={{
                      ...getColumnPinningStyle({ column: header.column }),
                    }}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {rows?.length ? (
              rows.map((row) => {
                const rowId = getRowId?.(row.original)
                const selected =
                  selectedRowId != null && rowId === selectedRowId
                return (
                  <TableRow
                    key={row.id}
                    data-state={selected && 'selected'}
                    className={cn(
                      onRowClick && 'cursor-pointer',
                      selected && 'bg-muted/50'
                    )}
                    onClick={
                      onRowClick
                        ? () => onRowClick(row.original as TData)
                        : undefined
                    }
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        style={{
                          ...getColumnPinningStyle({ column: cell.column }),
                        }}
                      >
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext()
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                )
              })
            ) : (
              <TableRow>
                <TableCell
                  colSpan={table.getAllColumns().length}
                  className='h-24 p-0 text-center'
                >
                  {emptyState}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      {hidePagination ? null : <DataTablePagination table={table} />}
      {actionBar &&
        table.getFilteredSelectedRowModel().rows.length > 0 &&
        actionBar}
    </div>
  )
}

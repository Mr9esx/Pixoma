import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createColumnHelper,
  type ColumnDef,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Pencil, RotateCcw, Save } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import {
  listTextTemplates,
  saveTextTemplates,
  type TextTemplate,
  textTemplateGroups,
} from '@/lib/api/text-templates'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { DataTableColumnHeader } from '@/components/data-table/column-header'
import { DataTable } from '@/components/data-table/data-table'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { PixomaLoading } from '@/components/feedback/pixoma-loading'
import { LongText } from '@/components/long-text'

// Edits the Telegram copy templates for a scope. channelId "" edits the
// platform default (Settings → 默认文案); any other id edits that channel's copy.
export function TextTemplatesEditor({
  channelId = '',
}: {
  channelId?: string
}) {
  const { t } = useTranslation()
  const q = useQuery({
    queryKey: queryKeys.textTemplates.channel(channelId),
    queryFn: () => listTextTemplates(channelId),
  })
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const data = useMemo(() => q.data ?? [], [q.data])
  const columnHelper = useMemo(() => createColumnHelper<TextTemplate>(), [])
  const editing = useMemo(
    () => data.find((item) => item.key === editingKey) ?? null,
    [data, editingKey]
  )

  const columns = useMemo<ColumnDef<TextTemplate, unknown>[]>(() => {
    return [
      columnHelper.accessor('key', {
        id: 'key',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('textTemplates.columnKey')}
          />
        ),
        cell: ({ row }) => (
          <div className='flex min-w-0 flex-col gap-1'>
            <span className='font-mono text-xs font-semibold'>
              {row.original.key}
            </span>
            <LongText className='max-w-64 text-xs text-muted-foreground'>
              {row.original.description}
            </LongText>
          </div>
        ),
      }),
      columnHelper.accessor('variables', {
        id: 'variables',
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('textTemplates.variablesLabel')}
          />
        ),
        cell: ({ row }) => {
          const variables = row.original.variables ?? []
          return variables.length > 0 ? (
            <div className='flex max-w-52 flex-wrap gap-1'>
              {variables.map((v) => (
                <Badge key={v} variant='outline' className='font-mono text-xs'>
                  {`{{ ${v} }}`}
                </Badge>
              ))}
            </div>
          ) : (
            <span className='text-xs text-muted-foreground'>
              {t('textTemplates.noVariables')}
            </span>
          )
        },
      }),
      columnHelper.accessor('default', {
        id: 'default',
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('textTemplates.defaultLabel')}
          />
        ),
        cell: ({ row }) => (
          <LongText className='max-w-64 font-mono text-xs text-muted-foreground'>
            {row.original.default}
          </LongText>
        ),
      }),
      columnHelper.accessor('value', {
        id: 'current',
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('textTemplates.currentLabel')}
          />
        ),
        cell: ({ row }) => {
          const value = row.original.value || ''
          return value ? (
            <LongText className='max-w-64 text-xs'>{value}</LongText>
          ) : (
            <span className='text-xs text-muted-foreground'>
              {t('textTemplates.usesDefault')}
            </span>
          )
        },
      }),
      columnHelper.display({
        id: 'actions',
        size: 110,
        enablePinning: true,
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('textTemplates.actions')}
          />
        ),
        cell: ({ row }) => (
          <Button
            variant='ghost'
            size='sm'
            onClick={() => setEditingKey(row.original.key)}
            title={t('textTemplates.edit')}
          >
            <Pencil className='size-3.5' />
            {t('textTemplates.edit')}
          </Button>
        ),
      }),
    ]
  }, [columnHelper, t, setEditingKey])

  return (
    <div className='flex min-h-0 flex-col gap-3'>
      {q.isLoading ? <LoadingSkeleton rows={3} /> : null}
      {q.isError ? (
        <ErrorBanner
          message={errorMessage(q.error)}
          onRetry={() => void q.refetch()}
        />
      ) : null}
      {!q.isLoading && !q.isError ? (
        <>
          {textTemplateGroups.map((group) => {
            const groupData = data.filter((item) => item.group === group)
            if (groupData.length === 0) return null
            return (
              <section key={group} className='flex flex-col gap-3'>
                <h2 className='text-base font-semibold'>
                  {t(`textTemplates.groups.${group}`)}
                </h2>
                <TemplateGroupTable data={groupData} columns={columns} />
              </section>
            )
          })}
          {editing ? (
            <EditTemplateDialog
              key={editing.key}
              channelId={channelId}
              template={editing}
              onClose={() => setEditingKey(null)}
            />
          ) : null}
        </>
      ) : null}
    </div>
  )
}

function TemplateGroupTable({
  data,
  columns,
}: {
  data: TextTemplate[]
  columns: ColumnDef<TextTemplate, unknown>[]
}) {
  const { t } = useTranslation()
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    enableColumnPinning: true,
    initialState: {
      columnPinning: { right: ['actions'] },
    },
  })

  return (
    <DataTable
      table={table}
      hidePagination
      emptyState={
        <p className='text-sm text-muted-foreground'>
          {t('textTemplates.empty')}
        </p>
      }
    />
  )
}

function EditTemplateDialog({
  channelId,
  template,
  onClose,
}: {
  channelId: string
  template: TextTemplate
  onClose: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [value, setValue] = useState(template.value ?? '')
  const hasOverride = value.trim().length > 0

  const saveMutation = useMutation({
    meta: { handledError: true },
    mutationFn: (save: string) =>
      saveTextTemplates(channelId, { [template.key]: save }),
    onSuccess: () => {
      toast.success(t('textTemplates.saved'))
      void queryClient.invalidateQueries({
        queryKey: queryKeys.textTemplates.channel(channelId),
      })
      onClose()
    },
    onError: () => toast.error(t('textTemplates.saveFailed')),
  })

  const variables = template.variables ?? []

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className='max-h-[85vh] overflow-y-auto sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle className='font-mono'>{template.key}</DialogTitle>
          <DialogDescription>{template.description}</DialogDescription>
        </DialogHeader>
        <div className='flex flex-col gap-4'>
          {variables.length > 0 ? (
            <div className='flex flex-col gap-1.5'>
              <Label className='text-xs text-muted-foreground'>
                {t('textTemplates.variablesLabel')}
              </Label>
              <div className='flex flex-wrap gap-1'>
                {variables.map((v) => (
                  <Badge
                    key={v}
                    variant='outline'
                    className='font-mono text-xs'
                  >
                    {`{{ ${v} }}`}
                  </Badge>
                ))}
              </div>
            </div>
          ) : null}
          <div className='flex flex-col gap-1.5'>
            <Label className='text-xs text-muted-foreground'>
              {t('textTemplates.defaultLabel')}
            </Label>
            <p className='rounded-md bg-muted/50 p-3 font-mono text-sm whitespace-pre-wrap text-muted-foreground'>
              {template.default}
            </p>
          </div>
          <div className='flex flex-col gap-1.5'>
            <Label className='text-xs text-muted-foreground'>
              {t('textTemplates.currentLabel')}
            </Label>
            <Textarea
              value={value}
              rows={4}
              disabled={saveMutation.isPending}
              placeholder={template.default}
              onChange={(e) => setValue(e.target.value)}
              className='resize-y font-mono text-sm'
            />
          </div>
        </div>
        <DialogFooter className='gap-2'>
          <Button
            type='button'
            variant='ghost'
            disabled={saveMutation.isPending || !hasOverride}
            onClick={() => setValue('')}
            title={t('textTemplates.resetHint')}
          >
            <RotateCcw className='size-3.5' />
            {t('textTemplates.reset')}
          </Button>
          <Button
            type='button'
            disabled={saveMutation.isPending}
            onClick={() => saveMutation.mutate(value)}
          >
            {saveMutation.isPending ? (
              <PixomaLoading />
            ) : (
              <Save className='size-3.5' />
            )}
            {saveMutation.isPending
              ? t('textTemplates.saving')
              : t('textTemplates.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

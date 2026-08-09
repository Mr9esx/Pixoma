import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { EmptyState } from '@/components/feedback/empty-state'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { listCases } from '@/lib/api/cases'
import { queryKeys } from '@/lib/api/query-keys'
import {
  getTgMenu,
  putTgMenu,
  type MenuAction,
  type MenuItem,
} from '@/lib/api/tg-menu'
import { cn } from '@/lib/utils'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function emptyItem(): MenuItem {
  return {
    id: `btn-${Date.now()}`,
    label: '',
    row: 0,
    col: 0,
    enabled: true,
    action: 'placeholder',
  }
}

function normalizeItem(item: MenuItem): MenuItem {
  const next: MenuItem = {
    id: item.id,
    label: item.label,
    row: Number(item.row) || 0,
    col: Number(item.col) || 0,
    enabled: Boolean(item.enabled),
    action: item.action,
  }
  switch (item.action) {
    case 'open_case':
      next.case_id = item.case_id?.trim() || undefined
      break
    case 'list_cases_by_tag':
      next.tag = item.tag?.trim() || undefined
      break
    case 'placeholder':
      next.placeholder_text = item.placeholder_text?.trim() || undefined
      break
    case 'reply_media': {
      const text = item.reply?.text?.trim() || undefined
      const images = (item.reply?.images ?? [])
        .map((u) => u.trim())
        .filter(Boolean)
      next.reply = { text, images: images.length ? images : undefined }
      break
    }
  }
  return next
}

function actionLabelKey(action: MenuAction): string {
  switch (action) {
    case 'open_case':
      return 'tgMenu.actionOpenCase'
    case 'list_cases_by_tag':
      return 'tgMenu.actionListByTag'
    case 'placeholder':
      return 'tgMenu.actionPlaceholder'
    case 'reply_media':
      return 'tgMenu.actionReplyMedia'
  }
}

export function TgMenuEditor() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [items, setItems] = useState<MenuItem[]>([])
  const [updatedAt, setUpdatedAt] = useState<string>('')
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null)

  const menuQuery = useQuery({
    queryKey: queryKeys.tgMenu.all,
    queryFn: getTgMenu,
  })

  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }] as const,
    queryFn: () => listCases({ limit: 200 }),
  })

  useEffect(() => {
    if (menuQuery.data) {
      setItems(structuredClone(menuQuery.data.items))
      setUpdatedAt(menuQuery.data.updated_at)
      setSelectedIndex((prev) => {
        if (menuQuery.data.items.length === 0) return null
        if (prev == null || prev >= menuQuery.data.items.length) return 0
        return prev
      })
    }
  }, [menuQuery.data])

  const saveMutation = useMutation({
    mutationFn: () => putTgMenu(items.map(normalizeItem)),
    onSuccess: (doc) => {
      setItems(structuredClone(doc.items))
      setUpdatedAt(doc.updated_at)
      void queryClient.invalidateQueries({ queryKey: queryKeys.tgMenu.all })
      toast.success(t('common.successSaved'))
    },
  })

  function updateItem(index: number, patch: Partial<MenuItem>) {
    setItems((prev) =>
      prev.map((it, i) => (i === index ? { ...it, ...patch } : it)),
    )
  }

  function setAction(index: number, action: MenuAction) {
    setItems((prev) =>
      prev.map((it, i) => {
        if (i !== index) return it
        const next: MenuItem = {
          id: it.id,
          label: it.label,
          row: it.row,
          col: it.col,
          enabled: it.enabled,
          action,
        }
        if (action === 'reply_media') {
          next.reply = { text: '', images: [] }
        }
        return next
      }),
    )
  }

  function addItem() {
    setItems((prev) => {
      const next = [...prev, emptyItem()]
      setSelectedIndex(next.length - 1)
      return next
    })
  }

  function removeSelected() {
    if (selectedIndex == null || items.length <= 1) return
    setItems((prev) => {
      const next = prev.filter((_, i) => i !== selectedIndex)
      setSelectedIndex(Math.min(selectedIndex, next.length - 1))
      return next
    })
  }

  if (menuQuery.isLoading) {
    return (
      <div className='min-h-0 flex-1 overflow-hidden rounded-md border p-4'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  if (menuQuery.isError) {
    return (
      <div className='min-h-0 flex-1 space-y-3 overflow-auto rounded-md border p-4'>
        <ErrorBanner
          message={errorMessage(menuQuery.error) ?? t('common.errorGeneric')}
          onRetry={() => void menuQuery.refetch()}
        />
      </div>
    )
  }

  const selected =
    selectedIndex != null && selectedIndex < items.length
      ? items[selectedIndex]
      : null

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='tg-menu-editor'
    >
      {saveMutation.isError ? (
        <ErrorBanner
          message={errorMessage(saveMutation.error) ?? t('common.errorGeneric')}
        />
      ) : null}

      <MasterDetailShell
        hasSelection={selected != null}
        onBackToList={() => setSelectedIndex(null)}
        list={
          <div className='flex h-full min-h-0 flex-col'>
            <div className='flex items-center justify-between gap-2 border-b px-4 py-3'>
              <div className='min-w-0'>
                <h2 className='text-sm font-semibold'>{t('tgMenu.listTitle')}</h2>
                {updatedAt ? (
                  <p className='truncate text-xs text-muted-foreground'>
                    {t('tgMenu.updatedAt')}: {updatedAt}
                  </p>
                ) : null}
              </div>
              <div className='flex shrink-0 items-center gap-2'>
                <Button type='button' size='sm' variant='outline' onClick={addItem}>
                  {t('tgMenu.addItem')}
                </Button>
                <Button
                  type='button'
                  size='sm'
                  onClick={() => saveMutation.mutate()}
                  disabled={saveMutation.isPending}
                >
                  {t('common.save')}
                </Button>
              </div>
            </div>

            {items.length === 0 ? (
              <EmptyState message={t('common.empty')} />
            ) : (
              <ul className='min-h-0 flex-1 overflow-auto'>
                {items.map((item, index) => {
                  const active = selectedIndex === index
                  return (
                    <li key={`${item.id}-${index}`}>
                      <button
                        type='button'
                        onClick={() => setSelectedIndex(index)}
                        className={cn(
                          'block w-full border-b px-4 py-3 text-left transition-colors',
                          active ? 'bg-muted' : 'hover:bg-muted/50',
                        )}
                      >
                        <div className='flex items-center justify-between gap-2'>
                          <span className='truncate text-sm font-medium'>
                            {item.label.trim() || t('tgMenu.untitled')}
                          </span>
                          <span
                            className={cn(
                              'shrink-0 rounded-sm px-1.5 py-0.5 text-[10px]',
                              item.enabled
                                ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
                                : 'bg-muted text-muted-foreground',
                            )}
                          >
                            {item.enabled
                              ? t('tgMenu.enabledShort')
                              : t('tgMenu.disabledShort')}
                          </span>
                        </div>
                        <p className='mt-1 truncate text-xs text-muted-foreground'>
                          {t(actionLabelKey(item.action))} · r{item.row}/c
                          {item.col}
                        </p>
                      </button>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
        }
        detail={
          selected && selectedIndex != null ? (
            <ItemEditor
              item={selected}
              index={selectedIndex}
              canRemove={items.length > 1}
              caseOptions={casesQuery.data ?? []}
              onUpdate={updateItem}
              onAction={setAction}
              onRemove={removeSelected}
            />
          ) : (
            <EmptyState message={t('common.selectItem')} />
          )
        }
      />
    </div>
  )
}

function ItemEditor({
  item,
  index,
  canRemove,
  caseOptions,
  onUpdate,
  onAction,
  onRemove,
}: {
  item: MenuItem
  index: number
  canRemove: boolean
  caseOptions: { id: string; name: string }[]
  onUpdate: (index: number, patch: Partial<MenuItem>) => void
  onAction: (index: number, action: MenuAction) => void
  onRemove: () => void
}) {
  const { t } = useTranslation()

  return (
    <div className='space-y-4' data-testid='tg-menu-item-editor'>
      <div className='flex items-start justify-between gap-3'>
        <div>
          <h2 className='text-lg font-semibold'>
            {item.label.trim() || t('tgMenu.untitled')}
          </h2>
          <p className='text-sm text-muted-foreground'>{t('tgMenu.detailHeading')}</p>
        </div>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          disabled={!canRemove}
          onClick={onRemove}
        >
          {t('tgMenu.removeItem')}
        </Button>
      </div>

      <div className='grid gap-3 sm:grid-cols-2'>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-id-${index}`}>{t('tgMenu.fieldId')}</Label>
          <Input
            id={`tg-menu-id-${index}`}
            value={item.id}
            onChange={(e) => onUpdate(index, { id: e.target.value })}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-label-${index}`}>
            {t('tgMenu.fieldLabel')}
          </Label>
          <Input
            id={`tg-menu-label-${index}`}
            value={item.label}
            onChange={(e) => onUpdate(index, { label: e.target.value })}
            autoComplete='off'
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-row-${index}`}>{t('tgMenu.fieldRow')}</Label>
          <Input
            id={`tg-menu-row-${index}`}
            type='number'
            value={item.row}
            onChange={(e) => onUpdate(index, { row: Number(e.target.value) })}
          />
        </div>
        <div className='space-y-1.5'>
          <Label htmlFor={`tg-menu-col-${index}`}>{t('tgMenu.fieldCol')}</Label>
          <Input
            id={`tg-menu-col-${index}`}
            type='number'
            value={item.col}
            onChange={(e) => onUpdate(index, { col: Number(e.target.value) })}
          />
        </div>
      </div>

      <div className='flex flex-wrap items-end gap-4'>
        <div className='flex items-center justify-between gap-3 rounded-md border px-3 py-2'>
          <Label htmlFor={`tg-menu-enabled-${index}`}>
            {t('tgMenu.fieldEnabled')}
          </Label>
          <Switch
            id={`tg-menu-enabled-${index}`}
            checked={item.enabled}
            onCheckedChange={(checked) =>
              onUpdate(index, { enabled: checked })
            }
          />
        </div>
        <div className='min-w-56 flex-1 space-y-1.5'>
          <Label>{t('tgMenu.fieldAction')}</Label>
          <Select
            value={item.action}
            onValueChange={(v) => onAction(index, v as MenuAction)}
          >
            <SelectTrigger className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='open_case'>
                {t('tgMenu.actionOpenCase')}
              </SelectItem>
              <SelectItem value='list_cases_by_tag'>
                {t('tgMenu.actionListByTag')}
              </SelectItem>
              <SelectItem value='placeholder'>
                {t('tgMenu.actionPlaceholder')}
              </SelectItem>
              <SelectItem value='reply_media'>
                {t('tgMenu.actionReplyMedia')}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {item.action === 'open_case' ? (
        <div className='space-y-1.5'>
          <Label>{t('tgMenu.fieldCaseId')}</Label>
          <Select
            value={item.case_id ?? ''}
            onValueChange={(v) => onUpdate(index, { case_id: v })}
          >
            <SelectTrigger className='w-full max-w-md'>
              <SelectValue placeholder={t('tgMenu.fieldCaseId')} />
            </SelectTrigger>
            <SelectContent>
              {caseOptions.map((c) => (
                <SelectItem key={c.id} value={c.id}>
                  {c.name} ({c.id})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      ) : null}

      {item.action === 'list_cases_by_tag' ? (
        <div className='max-w-md space-y-1.5'>
          <Label htmlFor={`tg-menu-tag-${index}`}>{t('tgMenu.fieldTag')}</Label>
          <Input
            id={`tg-menu-tag-${index}`}
            value={item.tag ?? ''}
            onChange={(e) => onUpdate(index, { tag: e.target.value })}
            autoComplete='off'
          />
        </div>
      ) : null}

      {item.action === 'placeholder' ? (
        <div className='max-w-xl space-y-1.5'>
          <Label htmlFor={`tg-menu-placeholder-${index}`}>
            {t('tgMenu.fieldPlaceholderText')}
          </Label>
          <Input
            id={`tg-menu-placeholder-${index}`}
            value={item.placeholder_text ?? ''}
            onChange={(e) =>
              onUpdate(index, { placeholder_text: e.target.value })
            }
            autoComplete='off'
          />
        </div>
      ) : null}

      {item.action === 'reply_media' ? (
        <div className='grid gap-3 md:grid-cols-2'>
          <div className='space-y-1.5'>
            <Label htmlFor={`tg-menu-reply-text-${index}`}>
              {t('tgMenu.fieldReplyText')}
            </Label>
            <Textarea
              id={`tg-menu-reply-text-${index}`}
              value={item.reply?.text ?? ''}
              onChange={(e) =>
                onUpdate(index, {
                  reply: { ...item.reply, text: e.target.value },
                })
              }
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor={`tg-menu-reply-images-${index}`}>
              {t('tgMenu.fieldReplyImages')}
            </Label>
            <Textarea
              id={`tg-menu-reply-images-${index}`}
              value={(item.reply?.images ?? []).join('\n')}
              onChange={(e) =>
                onUpdate(index, {
                  reply: {
                    ...item.reply,
                    images: e.target.value.split('\n'),
                  },
                })
              }
            />
          </div>
        </div>
      ) : null}
    </div>
  )
}

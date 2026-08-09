import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ErrorBanner } from '@/components/feedback/error-banner'
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

export function TgMenuEditor() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [items, setItems] = useState<MenuItem[]>([])
  const [updatedAt, setUpdatedAt] = useState<string>('')

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

  if (menuQuery.isLoading) {
    return <p className='text-muted-foreground text-sm'>{t('common.loading')}</p>
  }

  if (menuQuery.isError) {
    return (
      <ErrorBanner
        message={errorMessage(menuQuery.error) ?? t('common.errorGeneric')}
        onRetry={() => void menuQuery.refetch()}
      />
    )
  }

  return (
    <div className='space-y-4' data-testid='tg-menu-editor'>
      {saveMutation.isError ? (
        <ErrorBanner
          message={errorMessage(saveMutation.error) ?? t('common.errorGeneric')}
        />
      ) : null}

      <div className='flex flex-wrap items-center gap-3'>
        <Button
          type='button'
          onClick={() => saveMutation.mutate()}
          disabled={saveMutation.isPending}
        >
          {t('tgMenu.save')}
        </Button>
        <Button type='button' variant='outline' onClick={() => setItems((p) => [...p, emptyItem()])}>
          {t('tgMenu.addItem')}
        </Button>
        {updatedAt ? (
          <span className='text-muted-foreground text-xs'>
            {t('tgMenu.updatedAt')}: {updatedAt}
          </span>
        ) : null}
      </div>

      <div className='space-y-4'>
        {items.map((item, index) => (
          <div
            key={`${item.id}-${index}`}
            className='border-border space-y-3 rounded-md border p-4'
          >
            <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
              <div className='space-y-1'>
                <Label>{t('tgMenu.fieldId')}</Label>
                <Input
                  value={item.id}
                  onChange={(e) => updateItem(index, { id: e.target.value })}
                />
              </div>
              <div className='space-y-1'>
                <Label>{t('tgMenu.fieldLabel')}</Label>
                <Input
                  value={item.label}
                  onChange={(e) => updateItem(index, { label: e.target.value })}
                />
              </div>
              <div className='space-y-1'>
                <Label>{t('tgMenu.fieldRow')}</Label>
                <Input
                  type='number'
                  value={item.row}
                  onChange={(e) =>
                    updateItem(index, { row: Number(e.target.value) })
                  }
                />
              </div>
              <div className='space-y-1'>
                <Label>{t('tgMenu.fieldCol')}</Label>
                <Input
                  type='number'
                  value={item.col}
                  onChange={(e) =>
                    updateItem(index, { col: Number(e.target.value) })
                  }
                />
              </div>
            </div>

            <div className='flex flex-wrap items-center gap-4'>
              <div className='flex items-center gap-2'>
                <Switch
                  checked={item.enabled}
                  onCheckedChange={(checked) =>
                    updateItem(index, { enabled: checked })
                  }
                />
                <Label>{t('tgMenu.fieldEnabled')}</Label>
              </div>
              <div className='min-w-48 space-y-1'>
                <Label>{t('tgMenu.fieldAction')}</Label>
                <Select
                  value={item.action}
                  onValueChange={(v) => setAction(index, v as MenuAction)}
                >
                  <SelectTrigger>
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
              <Button
                type='button'
                variant='ghost'
                disabled={items.length <= 1}
                onClick={() =>
                  setItems((prev) => prev.filter((_, i) => i !== index))
                }
              >
                {t('tgMenu.removeItem')}
              </Button>
            </div>

            {item.action === 'open_case' ? (
              <div className='max-w-md space-y-1'>
                <Label>{t('tgMenu.fieldCaseId')}</Label>
                <Select
                  value={item.case_id ?? ''}
                  onValueChange={(v) => updateItem(index, { case_id: v })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder={t('tgMenu.fieldCaseId')} />
                  </SelectTrigger>
                  <SelectContent>
                    {(casesQuery.data ?? []).map((c) => (
                      <SelectItem key={c.id} value={c.id}>
                        {c.name} ({c.id})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}

            {item.action === 'list_cases_by_tag' ? (
              <div className='max-w-md space-y-1'>
                <Label>{t('tgMenu.fieldTag')}</Label>
                <Input
                  value={item.tag ?? ''}
                  onChange={(e) => updateItem(index, { tag: e.target.value })}
                />
              </div>
            ) : null}

            {item.action === 'placeholder' ? (
              <div className='max-w-xl space-y-1'>
                <Label>{t('tgMenu.fieldPlaceholderText')}</Label>
                <Input
                  value={item.placeholder_text ?? ''}
                  onChange={(e) =>
                    updateItem(index, { placeholder_text: e.target.value })
                  }
                />
              </div>
            ) : null}

            {item.action === 'reply_media' ? (
              <div className='grid gap-3 md:grid-cols-2'>
                <div className='space-y-1'>
                  <Label>{t('tgMenu.fieldReplyText')}</Label>
                  <Textarea
                    value={item.reply?.text ?? ''}
                    onChange={(e) =>
                      updateItem(index, {
                        reply: { ...item.reply, text: e.target.value },
                      })
                    }
                  />
                </div>
                <div className='space-y-1'>
                  <Label>{t('tgMenu.fieldReplyImages')}</Label>
                  <Textarea
                    value={(item.reply?.images ?? []).join('\n')}
                    onChange={(e) =>
                      updateItem(index, {
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
        ))}
      </div>
    </div>
  )
}

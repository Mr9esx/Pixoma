import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import {
  getChannelMenuExtras,
  putChannelMenuExtras,
  type MenuExtra,
} from '@/lib/api/channel-menu'
import { queryKeys } from '@/lib/api/query-keys'

export function ExtrasEditor({ channelId }: { channelId: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<Record<string, MenuExtra[]>>({})

  const extrasQuery = useQuery({
    queryKey: queryKeys.channels.extras(channelId),
    queryFn: () => getChannelMenuExtras(channelId),
  })

  useEffect(() => {
    if (extrasQuery.data) {
      setDraft(structuredClone(extrasQuery.data))
    }
  }, [extrasQuery.data])

  if (extrasQuery.isLoading) return <LoadingSkeleton rows={3} />
  if (extrasQuery.isError) {
    return <ErrorBanner message={String(extrasQuery.error)} />
  }
  const saveMutation = useMutation({
    mutationFn: () => putChannelMenuExtras(channelId, draft),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.extras(channelId) })
      toast.success(t('common.successSaved'))
    },
  })

  function addExtra() {
    const itemId = window.prompt(t('extras.itemPrompt'))?.trim()
    if (!itemId) return
    const type = window.prompt(t('extras.typePrompt'))?.trim() || 'tg_root_layout'
    setDraft((prev) => ({
      ...prev,
      [itemId]: [
        ...(prev[itemId] ?? []),
        {
          channel_id: channelId,
          menu_item_id: itemId,
          extra_type: type,
          extra_json: '{"columns":2}',
          updated_at: '',
        },
      ],
    }))
  }

  return (
    <div className='max-w-2xl space-y-4'>
      <p className='text-sm text-muted-foreground'>{t('extras.description')}</p>
      {Object.entries(draft).length === 0 && (
        <p className='text-sm text-muted-foreground'>{t('extras.empty')}</p>
      )}
      {Object.entries(draft).map(([itemId, list]) => (
        <div key={itemId} className='space-y-2 rounded-md border p-3'>
          <Label>{t('extras.item')}: {itemId}</Label>
          {list.map((extra, idx) => (
            <div key={`${extra.extra_type}-${idx}`} className='grid gap-2'>
              <Input
                value={extra.extra_type}
                onChange={(e) => {
                  const next = structuredClone(draft)
                  next[itemId][idx] = { ...extra, extra_type: e.target.value }
                  setDraft(next)
                }}
              />
              <Textarea
                value={extra.extra_json}
                onChange={(e) => {
                  const next = structuredClone(draft)
                  next[itemId][idx] = { ...extra, extra_json: e.target.value }
                  setDraft(next)
                }}
              />
            </div>
          ))}
        </div>
      ))}
      <div className='flex gap-2'>
        <Button variant='outline' onClick={addExtra}>
          {t('extras.add')}
        </Button>
        <Button disabled={saveMutation.isPending} onClick={() => saveMutation.mutate()}>
          {t('common.save')}
        </Button>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  deleteChannel,
  getChannel,
  setChannelEnabled,
  updateChannel,
} from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { MenuCardEditor } from '@/features/menu/menu-card-editor'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function ChannelDetailPanel({ id }: { id: string }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [token, setToken] = useState('')

  const channelQuery = useQuery({
    queryKey: queryKeys.channels.detail(id),
    queryFn: () => getChannel(id),
  })
  const ch = channelQuery.data

  const updateMutation = useMutation({
    mutationFn: () =>
      updateChannel(id, {
        name: name.trim() || undefined,
        token: token.trim() || undefined,
      }),
    onSuccess: () => {
      setToken('')
      void queryClient.invalidateQueries({
        queryKey: queryKeys.channels.detail(id),
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast.success(t('common.successSaved'))
    },
  })

  const enableMutation = useMutation({
    mutationFn: (enabled: boolean) => setChannelEnabled(id, enabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.channels.detail(id),
      })
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteChannel(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast.success(t('channels.deleted'))
    },
  })

  if (channelQuery.isLoading) return <LoadingSkeleton rows={8} />
  if (channelQuery.isError || !ch) {
    return (
      <ErrorBanner
        message={errorMessage(channelQuery.error) ?? t('common.errorGeneric')}
      />
    )
  }

  return (
    <div className='space-y-6 p-4' data-testid='channel-detail-panel'>
      <div className='flex items-start justify-between gap-3'>
        <div>
          <h2 className='text-lg font-semibold'>{ch.name}</h2>
          <p className='text-sm text-muted-foreground'>
            {ch.platform} · {ch.token_masked}
          </p>
        </div>
        <div className='flex shrink-0 gap-2'>
          <Button
            variant='outline'
            onClick={() => enableMutation.mutate(!ch.enabled)}
            disabled={enableMutation.isPending}
          >
            {ch.enabled ? t('channels.disable') : t('channels.enable')}
          </Button>
          <Button
            variant='destructive'
            disabled={ch.enabled || deleteMutation.isPending}
            onClick={() => deleteMutation.mutate()}
          >
            {t('channels.delete')}
          </Button>
        </div>
      </div>

      {updateMutation.isError ? (
        <ErrorBanner message={errorMessage(updateMutation.error)} />
      ) : null}
      {deleteMutation.isError ? (
        <ErrorBanner message={errorMessage(deleteMutation.error)} />
      ) : null}

      <section>
        <h3 className='mb-2 text-sm font-semibold'>{t('channels.tabBasic')}</h3>
        <Card className='max-w-xl'>
          <CardContent className='space-y-4 pt-6'>
            <div className='space-y-1.5'>
              <Label>{t('channels.name')}</Label>
              <Input
                value={name || ch.name}
                onChange={(e) => setName(e.target.value)}
                autoComplete='off'
              />
            </div>
            <div className='space-y-1.5'>
              <Label>{t('channels.token')}</Label>
              <Input
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={ch.token_masked}
                autoComplete='off'
              />
              <p className='text-xs text-muted-foreground'>
                {t('channels.tokenHint')}
              </p>
            </div>
            <Button
              disabled={
                updateMutation.isPending || (!name.trim() && !token.trim())
              }
              onClick={() => updateMutation.mutate()}
            >
              {t('common.save')}
            </Button>
          </CardContent>
        </Card>
      </section>

      <section>
        <h3 className='mb-2 text-sm font-semibold'>{t('channels.tabMenu')}</h3>
        <div className='min-h-[480px] rounded-md border p-4'>
          <MenuCardEditor channelId={id} />
        </div>
      </section>
    </div>
  )
}

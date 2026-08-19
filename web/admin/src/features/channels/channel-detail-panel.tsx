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
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { kit } from '@/features/edges/kit-classes'
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
    <section className={kit.pageSection} data-testid='channel-detail-panel'>
      <div className='flex min-w-0 flex-col gap-[6px]'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='flex min-w-0 flex-wrap items-center gap-2'>
            <h2 className={kit.title}>{ch.name}</h2>
            <span
              className={cn(
                'inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium',
                ch.enabled
                  ? 'border-emerald-600/20 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-900/30 dark:text-emerald-400'
                  : 'border-zinc-300 bg-zinc-50 text-zinc-700 dark:border-zinc-700 dark:bg-zinc-900/60 dark:text-zinc-300'
              )}
            >
              {ch.enabled ? t('channels.enabled') : t('channels.disabled')}
            </span>
          </div>
          <div className='flex shrink-0 flex-wrap gap-2'>
            <Button
              type='button'
              variant='outline'
              className={kit.btnGhost}
              onClick={() => enableMutation.mutate(!ch.enabled)}
              disabled={enableMutation.isPending}
            >
              {ch.enabled ? t('channels.disable') : t('channels.enable')}
            </Button>
            <Button
              type='button'
              variant='destructive'
              className='h-8 gap-1.5 rounded-md px-3 text-xs'
              disabled={ch.enabled || deleteMutation.isPending}
              onClick={() => deleteMutation.mutate()}
            >
              {t('channels.delete')}
            </Button>
          </div>
        </div>
        <p className={kit.desc}>
          {ch.platform} · {ch.token_masked}
        </p>
        <div className='mt-4 flex max-w-full flex-wrap items-center gap-2 text-xs'>
          <MetaChip label={t('channels.platform')} value={ch.platform} />
          <MetaChip label={t('channels.token')} value={ch.token_masked} />
          <MetaChip
            label={t('channels.fieldCreatedAt')}
            value={formatTime(ch.created_at)}
          />
          <MetaChip
            label={t('channels.fieldUpdatedAt')}
            value={formatTime(ch.updated_at)}
          />
        </div>
      </div>

      {updateMutation.isError ? (
        <ErrorBanner message={errorMessage(updateMutation.error)} />
      ) : null}
      {deleteMutation.isError ? (
        <ErrorBanner message={errorMessage(deleteMutation.error)} />
      ) : null}

      <section className='space-y-3'>
        <h3 className='text-sm font-semibold'>{t('channels.tabBasic')}</h3>
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

      <section className='space-y-3'>
        <h3 className='text-sm font-semibold'>{t('channels.tabMenu')}</h3>
        <div className='min-h-[480px] rounded-md border p-4'>
          <MenuCardEditor channelId={id} />
        </div>
      </section>
    </section>
  )
}

function MetaChip({ label, value }: { label: string; value: string }) {
  return (
    <span className='inline-flex items-center gap-1 rounded-md border px-2 py-1'>
      <span className='text-muted-foreground'>{label}</span>
      <span className='font-medium'>{value}</span>
    </span>
  )
}

function formatTime(iso: string): string {
  const ms = Date.parse(iso)
  if (!Number.isFinite(ms)) return iso
  return new Date(ms).toLocaleString()
}

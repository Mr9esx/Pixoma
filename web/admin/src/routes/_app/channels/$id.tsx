import { useState } from 'react'
import { createFileRoute, useParams } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ChannelMenuEditor } from '@/features/channels/channel-menu-editor'
import { ExtrasEditor } from '@/features/channels/extras-editor'
import {
  deleteChannel,
  getChannel,
  setChannelEnabled,
  updateChannel,
} from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'

export const Route = createFileRoute('/_app/channels/$id')({
  component: ChannelDetailPage,
})

function ChannelDetailPage() {
  const { id } = useParams({ from: '/_app/channels/$id' })
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
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.detail(id) })
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast.success(t('common.successSaved'))
    },
  })

  const enableMutation = useMutation({
    mutationFn: (enabled: boolean) => setChannelEnabled(id, enabled),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.detail(id) })
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

  if (channelQuery.isLoading) return <LoadingSkeleton rows={4} />
  if (channelQuery.isError || !ch) {
    return <ErrorBanner message={String(channelQuery.error)} />
  }

  return (
    <div data-layout='fixed' className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'>
      <div className='flex items-center justify-between'>
        <div>
          <h1 className='text-2xl font-bold tracking-tight'>{ch.name}</h1>
          <p className='text-sm text-muted-foreground'>
            {ch.platform} · {ch.token_masked}
          </p>
        </div>
        <div className='flex gap-2'>
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

      <Tabs defaultValue='basic' className='min-h-0 flex-1'>
        <TabsList>
          <TabsTrigger value='basic'>{t('channels.tabBasic')}</TabsTrigger>
          <TabsTrigger value='menu'>{t('channels.tabMenu')}</TabsTrigger>
          <TabsTrigger value='extras'>{t('channels.tabExtras')}</TabsTrigger>
        </TabsList>
        <TabsContent value='basic' className='mt-3'>
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
                <p className='text-xs text-muted-foreground'>{t('channels.tokenHint')}</p>
              </div>
              <Button
                disabled={updateMutation.isPending || (!name.trim() && !token.trim())}
                onClick={() => updateMutation.mutate()}
              >
                {t('common.save')}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value='menu' className='mt-3 h-full'>
          <ChannelMenuEditor channelId={id} />
        </TabsContent>
        <TabsContent value='extras' className='mt-3'>
          <ExtrasEditor channelId={id} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

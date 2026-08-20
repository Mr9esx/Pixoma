import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createChannel } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export const Route = createFileRoute('/_app/channels/new')({
  component: NewChannelPage,
})

function NewChannelPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [token, setToken] = useState('')

  const createMutation = useMutation({
    mutationFn: () => createChannel({ platform: 'telegram', name, token }),
    onSuccess: (ch) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast.success(t('channels.created'))
      void navigate({ to: '/channels/$id', params: { id: ch.id } })
    },
  })

  return (
    <div
      data-layout='fixed'
      className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'
    >
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('channels.new')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('channels.newDescription')}
        </p>
      </div>
      <Card className='max-w-xl'>
        <CardContent className='space-y-4 pt-6'>
          <div className='space-y-1.5'>
            <Label>{t('channels.platform')}</Label>
            <Input value='Telegram' disabled />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='channel-name'>{t('channels.name')}</Label>
            <Input
              id='channel-name'
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoComplete='off'
            />
          </div>
          <div className='space-y-1.5'>
            <Label htmlFor='channel-token'>{t('channels.token')}</Label>
            <Input
              id='channel-token'
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder='123456:ABC…'
              autoComplete='off'
            />
          </div>
          <Button
            disabled={!name.trim() || !token.trim() || createMutation.isPending}
            onClick={() => createMutation.mutate()}
          >
            {t('channels.create')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}

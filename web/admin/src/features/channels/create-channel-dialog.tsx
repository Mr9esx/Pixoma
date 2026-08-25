import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createChannel, type Channel } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Props = {
  onDone: (channel: Channel) => void
}

export function CreateChannelDialog({ onDone }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [token, setToken] = useState('')

  const createMutation = useMutation({
    mutationFn: () => createChannel({ platform: 'telegram', name, token }),
    onSuccess: (ch) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      toast.success(t('channels.created'))
      onDone(ch)
    },
  })

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t('channels.new')}</DialogTitle>
      </DialogHeader>
      <div className='space-y-4 py-4'>
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
      </div>
      <DialogFooter>
        <Button
          disabled={
            !name.trim() || !token.trim() || createMutation.isPending
          }
          onClick={() => createMutation.mutate()}
        >
          {t('channels.create')}
        </Button>
      </DialogFooter>
    </>
  )
}

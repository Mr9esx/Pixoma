import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createChannel, type Channel } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type Props = {
  onDone: (channel: Channel) => void
  onCancel: () => void
}

export function CreateChannelForm({ onDone, onCancel }: Props) {
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
    <div className='flex flex-1 flex-col gap-4'>
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
      <DialogFooter className='shrink-0'>
        <Button type='button' variant='outline' onClick={onCancel}>
          {t('common.cancel')}
        </Button>
        <Button
          disabled={!token.trim() || createMutation.isPending}
          onClick={() => createMutation.mutate()}
        >
          {t('channels.create')}
        </Button>
      </DialogFooter>
    </div>
  )
}

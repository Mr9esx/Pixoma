import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createChannel, type Channel } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { SecretInput } from '@/components/secret-input'

type Platform = 'telegram' | 'mcp'

type Props = {
  onDone: (channel: Channel) => void
  onCancel: () => void
}

export function CreateChannelForm({ onDone, onCancel }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [platform, setPlatform] = useState<Platform>('telegram')
  const [name, setName] = useState('')
  const [token, setToken] = useState('')

  const createMutation = useMutation({
    mutationFn: () =>
      createChannel(
        platform === 'mcp'
          ? { platform, name: name.trim() }
          : { platform, name: name.trim(), token }
      ),
    onSuccess: (ch) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.channels.all })
      void queryClient.invalidateQueries({ queryKey: queryKeys.linkHealth })
      toast.success(t('channels.created'))
      onDone(ch)
    },
  })

  const canSubmit =
    name.trim() !== '' &&
    (platform === 'mcp' || token.trim() !== '') &&
    !createMutation.isPending

  return (
    <div className='flex flex-1 flex-col gap-4'>
      <FieldGroup className='gap-4'>
        <Field>
          <FieldLabel htmlFor='channel-platform'>
            {t('channels.platform')}
          </FieldLabel>
          <Select
            value={platform}
            onValueChange={(value) => setPlatform(value as Platform)}
          >
            <SelectTrigger id='channel-platform' className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value='telegram'>
                  {t('channels.platformTelegram')}
                </SelectItem>
                <SelectItem value='mcp'>
                  {t('channels.platformMCP')}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>
        <Field>
          <FieldLabel htmlFor='channel-name'>{t('channels.name')}</FieldLabel>
          <Input
            id='channel-name'
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoComplete='off'
          />
        </Field>
        {platform === 'telegram' ? (
          <Field>
            <FieldLabel htmlFor='channel-token'>
              {t('channels.token')}
            </FieldLabel>
            <SecretInput
              id='channel-token'
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder='123456:ABC…'
              autoComplete='off'
            />
          </Field>
        ) : null}
      </FieldGroup>
      <DialogFooter className='shrink-0'>
        <Button type='button' variant='outline' onClick={onCancel}>
          {t('common.cancel')}
        </Button>
        <Button disabled={!canSubmit} onClick={() => createMutation.mutate()}>
          {t('channels.create')}
        </Button>
      </DialogFooter>
    </div>
  )
}

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createChannel, type Channel } from '@/lib/api/channels'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { RequiredBadge } from '@/components/required-badge'
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

type Platform = 'telegram' | 'mcp' | 'feishu'

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
  const [appId, setAppId] = useState('')
  const [appSecret, setAppSecret] = useState('')

  const createMutation = useMutation({
    mutationFn: () =>
      createChannel(
        platform === 'mcp'
          ? { platform, name: name.trim() }
          : platform === 'feishu'
            ? { platform, name: name.trim(), appId: appId.trim(), appSecret }
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
    (platform === 'mcp' ||
      (platform === 'feishu'
        ? appId.trim() !== '' && appSecret.trim() !== ''
        : token.trim() !== '')) &&
    !createMutation.isPending

  return (
    <div className='flex flex-1 flex-col gap-4'>
      <FieldGroup className='gap-4'>
        <Field>
          <FieldLabel htmlFor='channel-name'>
            {t('channels.name')}
            <RequiredBadge />
          </FieldLabel>
          <Input
            id='channel-name'
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoComplete='off'
            required
          />
        </Field>
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
                <SelectItem value='feishu'>
                  {t('channels.platformFeishu')}
                </SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>
        {platform === 'telegram' ? (
          <Field>
            <FieldLabel htmlFor='channel-token'>
              {t('channels.token')}
              <RequiredBadge />
            </FieldLabel>
            <SecretInput
              id='channel-token'
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder='123456:ABC…'
              autoComplete='off'
              required
            />
          </Field>
        ) : platform === 'feishu' ? (
          <>
            <Field>
              <FieldLabel htmlFor='channel-app-id'>
                {t('channels.appId')}
                <RequiredBadge />
              </FieldLabel>
              <Input
                id='channel-app-id'
                value={appId}
                onChange={(e) => setAppId(e.target.value)}
                placeholder='cli_…'
                autoComplete='off'
                required
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='channel-app-secret'>
                {t('channels.appSecret')}
                <RequiredBadge />
              </FieldLabel>
              <SecretInput
                id='channel-app-secret'
                value={appSecret}
                onChange={(e) => setAppSecret(e.target.value)}
                autoComplete='off'
                required
              />
              <p className='text-sm text-muted-foreground'>
                {t('channels.appIdHint')}
              </p>
            </Field>
          </>
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

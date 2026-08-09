import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  createInstance,
  patchInstance,
} from '@/lib/api/instances'
import { queryKeys } from '@/lib/api/query-keys'
import type { ComfyInstance } from '@/lib/api/types'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function parseCapabilities(raw: string): string[] {
  return raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

type CreateProps = {
  mode: 'create'
  initial?: undefined
}

type EditProps = {
  mode: 'edit'
  initial: ComfyInstance
}

type Props = CreateProps | EditProps

export function InstanceForm(props: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [id, setId] = useState(props.mode === 'edit' ? props.initial.id : '')
  const [baseUrl, setBaseUrl] = useState(
    props.mode === 'edit' ? props.initial.base_url : '',
  )
  const [enabled, setEnabled] = useState(
    props.mode === 'edit' ? props.initial.enabled : true,
  )
  const [capabilitiesRaw, setCapabilitiesRaw] = useState(
    props.mode === 'edit' ? props.initial.capabilities.join(', ') : '',
  )

  const createMutation = useMutation({
    mutationFn: createInstance,
    onSuccess: async (created) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.instances.all })
      toast.success(t('instances.createSuccess'))
      void navigate({
        to: '/instances/$instanceId',
        params: { instanceId: created.id },
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: (body: {
      base_url?: string
      enabled?: boolean
      capabilities?: string[]
    }) => {
      if (props.mode !== 'edit') {
        throw new Error('update requires edit mode')
      }
      return patchInstance(props.initial.id, body)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.instances.all })
      toast.success(t('common.successSaved'))
    },
  })

  const pending = createMutation.isPending || updateMutation.isPending
  const mutationError =
    createMutation.error ?? updateMutation.error ?? undefined

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const capabilities = parseCapabilities(capabilitiesRaw)
    if (props.mode === 'create') {
      const trimmedId = id.trim()
      if (!trimmedId || !baseUrl.trim()) return
      createMutation.mutate({
        id: trimmedId,
        base_url: baseUrl.trim(),
        enabled,
        capabilities,
      })
      return
    }
    updateMutation.mutate({
      base_url: baseUrl.trim(),
      enabled,
      capabilities,
    })
  }

  return (
    <form
      onSubmit={onSubmit}
      className='space-y-4'
      data-testid='instance-form'
    >
      <div className='space-y-2'>
        <Label htmlFor='instance-id'>{t('instances.fieldId')}</Label>
        <Input
          id='instance-id'
          value={id}
          onChange={(e) => setId(e.target.value)}
          disabled={props.mode === 'edit' || pending}
          required={props.mode === 'create'}
          autoComplete='off'
        />
      </div>

      <div className='space-y-2'>
        <Label htmlFor='instance-base-url'>{t('instances.fieldBaseUrl')}</Label>
        <Input
          id='instance-base-url'
          value={baseUrl}
          onChange={(e) => setBaseUrl(e.target.value)}
          disabled={pending}
          required
          autoComplete='off'
        />
      </div>

      <div className='flex items-center justify-between gap-3'>
        <Label htmlFor='instance-enabled'>{t('instances.fieldEnabled')}</Label>
        <Switch
          id='instance-enabled'
          checked={enabled}
          onCheckedChange={setEnabled}
          disabled={pending}
        />
      </div>

      <div className='space-y-2'>
        <Label htmlFor='instance-capabilities'>
          {t('instances.fieldCapabilities')}
        </Label>
        <Input
          id='instance-capabilities'
          value={capabilitiesRaw}
          onChange={(e) => setCapabilitiesRaw(e.target.value)}
          disabled={pending}
          placeholder={t('instances.capabilitiesPlaceholder')}
          autoComplete='off'
        />
        <p className='text-muted-foreground text-xs'>
          {t('instances.capabilitiesHint')}
        </p>
      </div>

      {mutationError ? (
        <ErrorBanner message={errorMessage(mutationError)} />
      ) : null}

      <div className='flex flex-wrap gap-2'>
        <Button type='submit' disabled={pending}>
          {props.mode === 'create' ? t('common.create') : t('common.save')}
        </Button>
        {props.mode === 'create' ? (
          <Button
            type='button'
            variant='outline'
            disabled={pending}
            onClick={() => void navigate({ to: '/instances' })}
          >
            {t('common.cancel')}
          </Button>
        ) : null}
      </div>
    </form>
  )
}

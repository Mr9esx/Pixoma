import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import { fetchPlatformSettings } from '@/lib/api/setup'
import type { ComfyEdge } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { PasswordInput } from '@/components/password-input'
import { edgeDeployCommand } from './deploy-command'

async function copyText(text: string) {
  await navigator.clipboard.writeText(text)
}

type Props = {
  edge: ComfyEdge
  tokenActions?: ReactNode
}

function isLoopbackURL(raw: string): boolean {
  try {
    const { hostname } = new URL(raw)
    return (
      hostname === 'localhost' ||
      hostname === '127.0.0.1' ||
      hostname === '::1' ||
      hostname === '0.0.0.0'
    )
  } catch {
    return false
  }
}

export function DeployCredentials({ edge, tokenActions }: Props) {
  const { t } = useTranslation()
  const token = edge.agent_token ?? ''

  const settingsQuery = useQuery({
    queryKey: queryKeys.settings.all,
    queryFn: fetchPlatformSettings,
  })
  const blobDriver = settingsQuery.data?.settings?.blob_driver || 'localfs'
  const serverPublicURL = settingsQuery.data?.public_url
  const controlPlaneURL =
    serverPublicURL && !isLoopbackURL(serverPublicURL)
      ? serverPublicURL
      : typeof window === 'undefined'
        ? ''
        : window.location.origin
  const commandInput = {
    controlPlaneURL,
    token,
    edgeId: edge.id,
    blobDriver,
  }
  const command = edgeDeployCommand(commandInput)
  const displayedCommand = edgeDeployCommand({
    ...commandInput,
    maskToken: true,
  })

  return (
    <div className='flex flex-col gap-6' data-testid='deploy-credentials'>
      <div className='flex flex-col gap-2'>
        <Label htmlFor='agent-token'>{t('edges.fieldAgentToken')}</Label>
        <div className='flex flex-wrap gap-2'>
          <PasswordInput
            id='agent-token'
            value={token}
            readOnly
            autoComplete='off'
            className='min-w-0 flex-1'
          />
          <Button
            type='button'
            variant='outline'
            disabled={!token}
            onClick={() => {
              void copyText(token).then(() => {
                toast.success(t('edges.copied'))
              })
            }}
          >
            {t('edges.copy')}
          </Button>
          {tokenActions}
        </div>
      </div>

      <div className='flex flex-col gap-2'>
        <Label htmlFor='edge-deploy'>{t('edges.deployCommand')}</Label>
        <pre
          id='edge-deploy'
          className='max-h-64 overflow-auto rounded-md bg-muted p-3 text-xs break-all whitespace-pre-wrap'
        >
          {displayedCommand}
        </pre>
        <Button
          type='button'
          variant='outline'
          className='w-full'
          disabled={!token}
          onClick={() => {
            void copyText(command).then(() => {
              toast.success(t('edges.copied'))
            })
          }}
        >
          {t('edges.copy')}
        </Button>
        <p className='text-xs text-muted-foreground'>
          {t('edges.deployComfyHint')}
        </p>
      </div>
    </div>
  )
}

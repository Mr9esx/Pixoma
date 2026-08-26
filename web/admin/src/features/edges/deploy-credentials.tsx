import { useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CheckIcon, ChevronsUpDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import { fetchPlatformSettings } from '@/lib/api/setup'
import { listTopics } from '@/lib/api/topics'
import type { ComfyEdge } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { PasswordInput } from '@/components/password-input'
import { edgeDeployCommand } from './deploy-command'

async function copyText(text: string) {
  await navigator.clipboard.writeText(text)
}

type Props = {
  edge: ComfyEdge
  tokenActions?: ReactNode
  /** 预选订阅的调度通道（默认取 edge.subscribe_topics）。 */
  initialSelectedTopics?: string[]
  /** 订阅通道变化时回调（用于向导把 Topic→节点 绑定落库）。 */
  onSubscribeTopicsChange?: (topics: string[]) => void
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

export function DeployCredentials({
  edge,
  tokenActions,
  initialSelectedTopics,
  onSubscribeTopicsChange,
}: Props) {
  const { t } = useTranslation()
  const token = edge.agent_token ?? ''
  const [selectedTopics, setSelectedTopics] = useState<string[]>(
    initialSelectedTopics ?? edge.subscribe_topics ?? []
  )

  const settingsQuery = useQuery({
    queryKey: queryKeys.settings.all,
    queryFn: fetchPlatformSettings,
  })
  const topicsQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
  })
  const enabledTopics = (topicsQuery.data ?? []).filter((tp) => tp.enabled)
  const blobDriver = settingsQuery.data?.settings?.blob_driver || 'localfs'
  const blob = settingsQuery.data?.settings
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
    subscribeTopics: selectedTopics,
    blobRoot: blob?.blob_root,
    blobEndpoint: blob?.blob_endpoint,
    blobRegion: blob?.blob_region,
    blobBucket: blob?.blob_bucket,
    blobAccessKey: blob?.blob_access_key,
    blobSecretKey: blob?.blob_secret_key,
  }
  const command = edgeDeployCommand(commandInput)
  const displayedCommand = edgeDeployCommand({
    ...commandInput,
    maskToken: true,
  })
  function toggleTopic(key: string) {
    commitTopics(
      selectedTopics.includes(key)
        ? selectedTopics.filter((k) => k !== key)
        : [...selectedTopics, key]
    )
  }
  function commitTopics(next: string[]) {
    setSelectedTopics(next)
    onSubscribeTopicsChange?.(next)
  }

  return (
    <div className='flex flex-col gap-6' data-testid='deploy-credentials'>
      <div className='flex flex-col gap-2'>
        <Label htmlFor='edge-deploy-topics'>{t('edges.deployTopics')}</Label>
        <Popover>
          <PopoverTrigger asChild>
            <Button
              id='edge-deploy-topics'
              type='button'
              variant='outline'
              role='combobox'
              className='w-full justify-between font-normal'
            >
              <span className='truncate'>
                {selectedTopics.length > 0
                  ? selectedTopics.join(', ')
                  : t('edges.deployTopicsPlaceholder')}
              </span>
              <ChevronsUpDown className='ml-2 size-4 shrink-0 opacity-50' />
            </Button>
          </PopoverTrigger>
          <PopoverContent
            className='w-(--radix-popover-trigger-width) p-0'
            align='start'
          >
            <Command>
              <CommandInput placeholder={t('edges.deployTopicsSearch')} />
              <CommandList>
                <CommandEmpty>{t('edges.deployTopicsEmpty')}</CommandEmpty>
                <CommandGroup>
                  {enabledTopics.map((tp) => {
                    const selected = selectedTopics.includes(tp.key)
                    return (
                      <CommandItem
                        key={tp.key}
                        value={tp.key}
                        onSelect={() => toggleTopic(tp.key)}
                      >
                        <div
                          className={cn(
                            'flex size-4 items-center justify-center rounded-sm border border-primary',
                            selected
                              ? 'bg-primary text-primary-foreground'
                              : 'opacity-50 [&_svg]:invisible'
                          )}
                        >
                          <CheckIcon className='size-3 text-background' />
                        </div>
                        <span className='font-medium'>{tp.key}</span>
                        {tp.name ? (
                          <span className='truncate text-xs text-muted-foreground'>
                            {tp.name}
                          </span>
                        ) : null}
                      </CommandItem>
                    )
                  })}
                </CommandGroup>
                {selectedTopics.length > 0 ? (
                  <>
                    <CommandSeparator />
                    <CommandGroup>
                      <CommandItem
                        onSelect={() => commitTopics([])}
                        className='justify-center text-center'
                      >
                        {t('edges.deployTopicsClear')}
                      </CommandItem>
                    </CommandGroup>
                  </>
                ) : null}
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
        <p className='text-xs text-muted-foreground'>
          {t('edges.deployTopicsHint')}
        </p>
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
          variant='default'
          className='w-full'
          disabled={!token}
          onClick={() => {
            void copyText(command).then(() => {
              toast.success(t('edges.copied'))
            })
          }}
        >
          {t('edges.copyDeployCommand')}
        </Button>
        <p className='text-xs text-muted-foreground'>
          {t('edges.deployComfyHint')}
        </p>
      </div>

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
    </div>
  )
}

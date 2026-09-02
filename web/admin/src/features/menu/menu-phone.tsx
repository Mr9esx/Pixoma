import type { ReactNode } from 'react'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { PenLine } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listCases } from '@/lib/api/cases'
import type {
  Media,
  MenuAction,
  MenuButton,
  MenuCard,
  MenuTree,
} from '@/lib/api/channel-menu'
import { resolveMediaKey } from '@/lib/api/media'
import { queryKeys } from '@/lib/api/query-keys'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { useMediaObjectUrl } from '@/features/cases/lib/use-media-object-url'
import { ListTasksPreview } from './list-tasks-preview'
import { toWorkflowRef } from './node-view'
import { WorkflowInfoCard } from './workflow-info-card'

export function MenuPhone({
  tree,
  channelId,
}: {
  tree: MenuTree
  channelId: string
}) {
  const { t } = useTranslation()
  const [stack, setStack] = useState<MenuCard[]>([])
  const [peek, setPeek] = useState<MenuAction | null>(null)
  const card = stack.length > 0 ? stack[stack.length - 1] : undefined
  const buttons: MenuButton[] = card ? (card.buttons ?? []) : tree.items
  const columns = tree.columns || 2
  const casesQuery = useQuery({
    queryKey: [...queryKeys.cases.all, { limit: 200 }],
    queryFn: () => listCases({ limit: 200 }),
  })
  const workflows = (casesQuery.data ?? []).map(toWorkflowRef)

  return (
    <div
      data-testid='menu-phone'
      className='flex h-[500px] w-full flex-col overflow-hidden rounded-[1.75rem] border bg-background'
    >
      <div
        data-testid='menu-phone-info'
        className='flex min-h-0 flex-1 flex-col overflow-hidden border-b bg-muted'
      >
        {stack.length > 0 ? (
          <div className='flex h-11 shrink-0 items-center px-3'>
            <Button
              type='button'
              variant='ghost'
              size='sm'
              className='h-11'
              onClick={() => {
                setStack((s) => s.slice(0, -1))
                setPeek(null)
              }}
            >
              {t('menu.backTo')}
            </Button>
          </div>
        ) : null}
        <div className='min-h-0 flex-1 overflow-y-auto p-3'>
          <PhoneInfo
            peek={peek}
            card={card}
            channelId={channelId}
            workflows={workflows}
          />
        </div>
      </div>
      <div
        data-testid='menu-phone-keys'
        className='flex h-[35%] min-h-0 shrink-0 flex-col gap-2 overflow-hidden p-3'
      >
        <div className='min-h-0 flex-1 overflow-y-auto'>
          <div
            className='grid gap-2'
            style={{
              gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
            }}
          >
            {buttons.map((b) => (
              <button
                key={b.id}
                type='button'
                data-testid='menu-phone-key'
                className='h-11 rounded-md border bg-secondary px-2 text-sm'
                onClick={() => {
                  if (b.action.type === 'open_card' && b.action.card) {
                    setStack((s) => [...s, b.action.card!])
                    setPeek(null)
                    return
                  }
                  setPeek(b.action)
                }}
              >
                {b.label}
              </button>
            ))}
          </div>
        </div>
        <Button asChild variant='outline' size='sm' className='w-full shrink-0'>
          <Link
            to='/channels/$id/menu'
            params={{ id: channelId }}
            data-testid='edit-menu'
          >
            <PenLine data-icon='inline-start' />
            {t('menu.editMenu')}
          </Link>
        </Button>
      </div>
    </div>
  )
}

function PhoneInfo({
  peek,
  card,
  channelId,
  workflows,
}: {
  peek: MenuAction | null
  card?: MenuCard
  channelId: string
  workflows: ReturnType<typeof toWorkflowRef>[]
}) {
  const { t } = useTranslation()
  if (peek) {
    return (
      <ActionInfo action={peek} channelId={channelId} workflows={workflows} />
    )
  }
  if (card) {
    return (
      <PhonePeekCard
        title={t('menu.actionOpenCard')}
        testId='menu-phone-card-text'
      >
        {card.text ? (
          <p className='m-0 text-sm whitespace-pre-wrap'>{card.text}</p>
        ) : (
          <PeekEmpty />
        )}
      </PhonePeekCard>
    )
  }
  return null
}

function ActionInfo({
  action,
  channelId,
  workflows,
}: {
  action: MenuAction
  channelId: string
  workflows: ReturnType<typeof toWorkflowRef>[]
}) {
  const { t } = useTranslation()
  if (action.type === 'open_workflow') {
    const wf = workflows.find(
      (w) => String(w.id) === String(action.workflow_id)
    )
    if (wf) {
      return (
        <div data-testid='menu-phone-workflow'>
          <WorkflowInfoCard workflow={wf} className='max-w-none' />
        </div>
      )
    }
    return (
      <PhonePeekCard
        title={t('menu.actionOpenWorkflow')}
        testId='menu-phone-workflow'
      >
        <PeekEmpty />
      </PhonePeekCard>
    )
  }
  if (action.type === 'list_tasks') {
    return (
      <PhonePeekCard
        title={t('menu.actionListTasks')}
        testId='menu-phone-tasks'
      >
        <ListTasksPreview
          channelId={channelId}
          className='rounded-none border-0 bg-transparent p-0'
        />
      </PhonePeekCard>
    )
  }
  if (action.type === 'open_url') {
    return (
      <PhonePeekCard title={t('menu.actionOpenUrl')} testId='menu-phone-url'>
        {action.url ? (
          <p className='m-0 text-sm break-all'>{action.url}</p>
        ) : (
          <PeekEmpty />
        )}
      </PhonePeekCard>
    )
  }
  if (action.type === 'send_text' || action.type === 'copy_text') {
    const title =
      action.type === 'copy_text'
        ? t('menu.actionCopyText')
        : t('menu.actionSendText')
    return (
      <PhonePeekCard title={title} testId='menu-phone-text'>
        {action.text ? (
          <p className='m-0 rounded-lg bg-muted px-3 py-2 text-sm whitespace-pre-wrap'>
            {action.text}
          </p>
        ) : (
          <PeekEmpty />
        )}
      </PhonePeekCard>
    )
  }
  if (action.type === 'send_media') {
    return (
      <PhonePeekCard
        title={t('menu.actionSendMedia')}
        testId='menu-phone-media'
      >
        {action.media?.length ? (
          <div className='flex flex-col gap-2'>
            {action.media.map((m, i) => (
              <PhoneMediaThumb key={`${m.url}-${i}`} media={m} />
            ))}
          </div>
        ) : (
          <PeekEmpty />
        )}
      </PhonePeekCard>
    )
  }
  if (action.type === 'open_card') {
    return (
      <PhonePeekCard
        title={t('menu.actionOpenCard')}
        testId='menu-phone-card-text'
      >
        {action.card?.text ? (
          <p className='m-0 text-sm whitespace-pre-wrap'>{action.card.text}</p>
        ) : (
          <PeekEmpty />
        )}
      </PhonePeekCard>
    )
  }
  return (
    <PhonePeekCard title={t('menu.actionRow')} testId='menu-phone-empty-action'>
      <PeekEmpty />
    </PhonePeekCard>
  )
}

function PeekEmpty() {
  const { t } = useTranslation()
  return (
    <p
      className='m-0 text-sm text-muted-foreground'
      data-testid='menu-phone-empty'
    >
      {t('menu.previewEmpty')}
    </p>
  )
}

function PhonePeekCard({
  title,
  testId,
  children,
}: {
  title: string
  testId: string
  children: ReactNode
}) {
  return (
    <Card className='gap-3 py-3' data-testid={testId}>
      <CardHeader className='px-3'>
        <Badge variant='outline'>{title}</Badge>
      </CardHeader>
      <CardContent className='px-3'>{children}</CardContent>
    </Card>
  )
}

function PhoneMediaThumb({ media }: { media: Media }) {
  const previewUrl = useMediaObjectUrl(media.url)
  const key = resolveMediaKey(media.url)
  const isVideo = /\.(mp4|webm)$/i.test(key ?? '')
  return (
    <div className='overflow-hidden rounded-lg bg-muted'>
      {previewUrl && isVideo ? (
        <video
          src={`${previewUrl}#t=0.1`}
          muted
          playsInline
          preload='metadata'
          disablePictureInPicture
          className='h-32 w-full object-cover'
        />
      ) : previewUrl ? (
        <img
          src={previewUrl}
          alt={media.caption || media.kind}
          className='h-32 w-full object-cover'
        />
      ) : (
        <p className='m-0 px-3 py-2 text-sm break-all'>{media.url}</p>
      )}
      {media.caption ? (
        <p className='m-0 px-3 py-2 text-sm'>{media.caption}</p>
      ) : null}
    </div>
  )
}

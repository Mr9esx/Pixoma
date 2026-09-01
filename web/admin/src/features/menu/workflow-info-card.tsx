import { ArrowRightIcon, BellIcon, SparklesIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { resolveMediaKey } from '@/lib/api/media'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useMediaObjectUrl } from '@/features/cases/lib/use-media-object-url'
import { type WorkflowRef } from './node-view'

/** 选中「开始工作流」且已绑定目标时，用 c-card-7 视觉展示该工作流信息。 */
export function WorkflowInfoCard({ workflow }: { workflow: WorkflowRef }) {
  const { t } = useTranslation()
  // Tag 只展示工作流自身的标签；没有就不渲染，避免用动作分组名（如“工作流能力”）误导。
  const badgeText = workflow.tags?.[0] ?? workflow.categories?.[0] ?? ''
  // preview 存的是鉴权媒体 key，需拉 Blob 转 objectURL 才能显示。
  const previewUrl = useMediaObjectUrl(workflow.preview)
  const previewKey = resolveMediaKey(workflow.preview)
  const previewIsVideo = /\.(mp4|webm)$/i.test(previewKey ?? '')

  return (
    <Card
      className='w-full max-w-xs gap-3 p-3'
      data-testid='workflow-info-card'
    >
      <CardContent className='flex flex-col gap-3 p-0'>
        <div className='relative h-44 w-full overflow-hidden rounded-lg bg-muted'>
          {previewUrl && previewIsVideo ? (
            <video
              src={`${previewUrl}#t=0.1`}
              muted
              playsInline
              preload='metadata'
              disablePictureInPicture
              className='h-full w-full object-cover'
            />
          ) : previewUrl ? (
            <img
              src={previewUrl}
              alt={workflow.name}
              width={1000}
              height={800}
              className='h-full w-full object-cover'
            />
          ) : (
            <div className='flex h-full w-full items-center justify-center'>
              <SparklesIcon
                aria-hidden='true'
                className='size-8 text-muted-foreground'
              />
            </div>
          )}
        </div>

        {badgeText ? (
          <div className='flex items-center gap-1.5'>
            <Badge variant='outline' className='text-sm'>
              <BellIcon aria-hidden='true' />
              {badgeText}
            </Badge>
          </div>
        ) : null}

        <div className='flex flex-col gap-1'>
          <p className='text-sm font-semibold text-foreground'>
            {workflow.name}
          </p>
          {workflow.description ? (
            <p className='line-clamp-2 text-sm text-muted-foreground'>
              {workflow.description}
            </p>
          ) : null}
        </div>

        <Button asChild size='sm' className='justify-between'>
          <a href={`/cases/${workflow.id}`} target='_blank' rel='noreferrer'>
            {t('menu.workflowCardView')}
            <ArrowRightIcon aria-hidden='true' className='size-4' />
          </a>
        </Button>
      </CardContent>
    </Card>
  )
}

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listTopics } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { DEFAULT_TOPIC_KEY } from '@/features/task-flow/types'
import { CreateTopicForm } from '@/features/topics/create-topic-form'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

export function Step2Queue({ shared, next, back }: Props) {
  const { t } = useTranslation()
  const [createOpen, setCreateOpen] = useState(false)
  const topicsQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
  })
  const topics = [...(topicsQuery.data ?? [])].sort((a, b) => {
    if (a.key === DEFAULT_TOPIC_KEY) return -1
    if (b.key === DEFAULT_TOPIC_KEY) return 1
    return a.key.localeCompare(b.key)
  })

  return (
    <WizardChrome
      step={2}
      onBack={() => back({})}
      onNext={() => next({})}
      nextLabel={t('quickConfig.next')}
      nextDisabled={!shared.topicKey}
    >
      <div className='space-y-3'>
        <div className='flex items-center justify-between gap-2'>
          <h3 className='text-sm font-semibold'>{t('quickConfig.queueStep')}</h3>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => setCreateOpen(true)}
          >
            <Plus className='size-4' />
            {t('quickConfig.newTopic')}
          </Button>
        </div>
        {topicsQuery.isLoading ? (
          <LoadingSkeleton rows={2} />
        ) : (
          <ul className='space-y-1'>
            {topics.map((topic) => (
              <li key={topic.key}>
                <button
                  type='button'
                  onClick={() => shared.updateTopic(topic.key, null)}
                  className={cn(
                    'flex w-full items-center justify-between rounded-md border border-border px-3 py-2.5 text-left',
                    shared.topicKey === topic.key &&
                      'border-primary bg-muted/60'
                  )}
                >
                  <span className='text-sm font-medium'>{topic.name}</span>
                  <span className='text-xs text-muted-foreground'>
                    {topic.key}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className='sm:max-w-[504px]'>
          <DialogHeader>
            <DialogTitle>{t('quickConfig.newTopic')}</DialogTitle>
          </DialogHeader>
          <CreateTopicForm
            onCancel={() => setCreateOpen(false)}
            onDone={(topic) => {
              shared.updateTopic(topic.key, null)
              setCreateOpen(false)
            }}
          />
        </DialogContent>
      </Dialog>
    </WizardChrome>
  )
}

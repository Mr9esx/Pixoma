import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createTopic, type Topic } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const KEY_PATTERN = /^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$/

type Props = {
  onDone: (topic: Topic) => void
  onCancel: () => void
}

export function CreateTopicForm({ onDone, onCancel }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [key, setKey] = useState('')
  const [name, setName] = useState('')

  const keyValid = KEY_PATTERN.test(key.trim())
  const canCreate = keyValid && Boolean(name.trim())

  const createMutation = useMutation({
    mutationFn: () => createTopic({ key: key.trim(), name: name.trim() }),
    onSuccess: (topic) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.topics.all })
      toast.success(t('topics.created'))
      onDone(topic)
    },
  })

  return (
    <div className='max-w-xl space-y-4'>
      <div className='space-y-1.5'>
        <Label htmlFor='topic-key'>{t('topics.fieldKey')}</Label>
        <Input
          id='topic-key'
          value={key}
          onChange={(e) => setKey(e.target.value)}
          placeholder='fast-gpu'
          autoComplete='off'
        />
        <p className='text-xs text-muted-foreground'>
          {t('topics.keyHint')}
        </p>
      </div>
      <div className='space-y-1.5'>
        <Label htmlFor='topic-name'>{t('topics.fieldName')}</Label>
        <Input
          id='topic-name'
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('topics.namePlaceholder')}
          autoComplete='off'
        />
      </div>
      {createMutation.isError ? (
        <p className='text-sm text-destructive'>
          {createMutation.error instanceof Error
            ? createMutation.error.message
            : t('common.errorGeneric')}
        </p>
      ) : null}
      <div className='sticky bottom-0 z-10 -mx-6 -mb-7 flex flex-wrap gap-2 border-t bg-card px-6 py-3 md:-mx-8 md:px-8'>
        <Button
          disabled={!canCreate || createMutation.isPending}
          onClick={() => createMutation.mutate()}
        >
          {t('topics.create')}
        </Button>
        <Button type='button' variant='outline' onClick={onCancel}>
          {t('common.cancel')}
        </Button>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import { createTopic, type Topic } from '@/lib/api/topics'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
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
    <div className='flex flex-1 flex-col gap-4'>
      <div className='space-y-1.5'>
        <Label htmlFor='topic-key'>{t('topics.fieldKey')}</Label>
        <Input
          id='topic-key'
          value={key}
          onChange={(e) => setKey(e.target.value)}
          placeholder='fast-gpu'
          autoComplete='off'
        />
        <p className='text-xs text-muted-foreground'>{t('topics.keyHint')}</p>
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
      <DialogFooter className='shrink-0'>
        <Button type='button' variant='outline' onClick={onCancel}>
          {t('common.cancel')}
        </Button>
        <Button
          disabled={!canCreate || createMutation.isPending}
          onClick={() => createMutation.mutate()}
        >
          {t('topics.create')}
        </Button>
      </DialogFooter>
    </div>
  )
}

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { TriangleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { createTopic, type Topic } from '@/lib/api/topics'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'

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
      void queryClient.invalidateQueries({ queryKey: queryKeys.linkHealth })
      onDone(topic)
    },
  })

  return (
    <div className='flex flex-1 flex-col gap-4'>
      <FieldGroup className='gap-4'>
        <Field data-invalid={!keyValid}>
          <FieldLabel htmlFor='topic-key'>{t('topics.fieldKey')}</FieldLabel>
          <Input
            id='topic-key'
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder='fast-gpu'
            autoComplete='off'
          />
          <FieldDescription className='text-xs'>
            {t('topics.keyHint')}
          </FieldDescription>
        </Field>
        <Field>
          <FieldLabel htmlFor='topic-name'>{t('topics.fieldName')}</FieldLabel>
          <Input
            id='topic-name'
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('topics.namePlaceholder')}
            autoComplete='off'
          />
        </Field>
      </FieldGroup>
      {createMutation.error ? (
        <Alert variant='destructive' className='px-3 py-2'>
          <TriangleAlert aria-hidden='true' />
          <AlertTitle>{t('quickConfig.saveFailed')}</AlertTitle>
          <AlertDescription>
            {createMutation.error instanceof Error
              ? createMutation.error.message
              : t('quickConfig.saveFailed')}
          </AlertDescription>
        </Alert>
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

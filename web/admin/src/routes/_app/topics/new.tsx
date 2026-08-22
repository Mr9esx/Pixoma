import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { createTopic } from '@/lib/api/topics'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export const Route = createFileRoute('/_app/topics/new')({
  component: NewTopicPage,
})

const KEY_PATTERN = /^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$/

function NewTopicPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
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
      void navigate({
        to: '/topics/$key',
        params: { key: topic.key },
      })
    },
  })

  return (
    <div
      data-layout='fixed'
      className='flex min-h-0 flex-1 flex-col gap-3 overflow-hidden'
    >
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('topics.new')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('topics.newDescription')}
        </p>
      </div>
      <Card className='max-w-xl'>
        <CardContent className='space-y-4 pt-6'>
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
          <Button
            disabled={!canCreate || createMutation.isPending}
            onClick={() => createMutation.mutate()}
          >
            {t('topics.create')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}

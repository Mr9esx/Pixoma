import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import { taskActionErrorMessage } from '@/lib/api/task-errors'
import { cancelTask, getTask } from '@/lib/api/tasks'
import type { TaskRecord } from '@/lib/api/types'
import { formatDateTime, formatUserLabel } from '@/lib/format'
import { Button } from '@/components/ui/button'
import { Reveal } from '@/components/ui/reveal'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { taskStatusLabelKey } from './list-panel'

type Props = {
  id: string
}

function Field({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className='grid gap-1 sm:grid-cols-[10rem_1fr] sm:items-start'>
      <dt className='text-xs font-medium text-muted-foreground'>{label}</dt>
      <dd className='text-sm break-all'>{value || '—'}</dd>
    </div>
  )
}

function TaskFields({
  task,
  t,
}: {
  task: TaskRecord
  t: (k: string) => string
}) {
  const statusKey = taskStatusLabelKey(task.status)
  return (
    <dl className='space-y-3'>
      <Field label={t('tasks.fieldId')} value={task.id} />
      <Field
        label={t('tasks.fieldStatus')}
        value={statusKey ? t(statusKey) : task.status}
      />
      <Field label={t('tasks.fieldCaseId')} value={task.case_id} />
      <Field label={t('tasks.fieldPlatform')} value={task.channel_name} />
      <Field
        label={t('tasks.fieldUser')}
        value={formatUserLabel(task.user, task.user_id)}
      />
      <Field label={t('tasks.fieldSessionId')} value={task.session_id} />
      <Field
        label={t('tasks.fieldChatId')}
        value={task.chat_id != null ? String(task.chat_id) : undefined}
      />
      <Field label={t('tasks.fieldEdgeId')} value={task.edge_id} />
      <Field label={t('tasks.fieldPromptId')} value={task.prompt_id} />
      <Field label={t('tasks.fieldErrorCode')} value={task.error_code} />
      <Field label={t('tasks.fieldErrorMessage')} value={task.error_message} />
      <Field
        label={t('tasks.fieldCreatedAt')}
        value={formatDateTime(task.created_at)}
      />
      <Field
        label={t('tasks.fieldUpdatedAt')}
        value={formatDateTime(task.updated_at)}
      />
    </dl>
  )
}

const cancellableStatuses = new Set(['pending', 'queued'])

export function TaskDetailPanel({ id }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const detailQuery = useQuery({
    queryKey: queryKeys.tasks.detail(id),
    queryFn: () => getTask(id),
  })

  const cancelMut = useMutation({
    mutationFn: () => cancelTask(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all })
      void queryClient.invalidateQueries({
        queryKey: queryKeys.tasks.detail(id),
      })
      toast.success(t('tasks.cancelSuccess'))
    },
    onError: (err) => {
      toast.error(taskActionErrorMessage(err, t))
    },
  })

  if (detailQuery.isLoading) {
    return (
      <div data-testid='task-detail-panel'>
        <LoadingSkeleton rows={8} />
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div data-testid='task-detail-panel' className='space-y-3'>
        <ErrorBanner
          message={taskActionErrorMessage(detailQuery.error, t)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }

  const task = detailQuery.data
  if (!task) return null

  return (
    <Reveal className='space-y-4' data-testid='task-detail-panel'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h2 className='text-lg font-semibold'>{task.id}</h2>
          <p className='text-sm text-muted-foreground'>
            {t('tasks.detailHeading')}
          </p>
        </div>
        {cancellableStatuses.has(task.status) ? (
          <Button
            type='button'
            variant='destructive'
            size='sm'
            disabled={cancelMut.isPending}
            onClick={() => cancelMut.mutate()}
          >
            {t('tasks.cancel')}
          </Button>
        ) : null}
      </div>
      <TaskFields task={task} t={t} />
    </Reveal>
  )
}

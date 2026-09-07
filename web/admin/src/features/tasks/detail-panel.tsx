import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CircleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { queryKeys } from '@/lib/api/query-keys'
import { taskActionErrorMessage } from '@/lib/api/task-errors'
import { cancelTask, getTask } from '@/lib/api/tasks'
import { formatDateTime, formatUserLabel } from '@/lib/format'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import {
  RelatedChip,
  ResourceDetailFallbackTitle,
  ResourceDetailLayout,
  type IdentifierItem,
} from '@/features/operations/detail-layout'
import {
  lifecycleBadgeClass,
  opsIdentity,
} from '@/features/operations/identity'
import type { OperationsDetailTarget } from '@/features/operations/types'
import { resourceDetailBodyClassName } from '@/features/resource-modal'
import { taskStatusLabelKey } from './list-panel'

type Props = {
  id: string
  onOpenRelated?: (target: OperationsDetailTarget) => void
}

const cancellableStatuses = new Set(['pending', 'queued'])

export function TaskDetailPanel({ id, onOpenRelated }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const detailQuery = useQuery({
    queryKey: queryKeys.tasks.detail(id),
    queryFn: () => getTask(id),
  })

  const cancelMut = useMutation({
    meta: { handledError: true },
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
      <>
        <ResourceDetailFallbackTitle>
          {t('tasks.detailHeading')}
        </ResourceDetailFallbackTitle>
        <div
          data-testid='task-detail-panel'
          className={resourceDetailBodyClassName}
        >
          <LoadingSkeleton rows={8} />
        </div>
      </>
    )
  }

  if (detailQuery.isError) {
    return (
      <>
        <ResourceDetailFallbackTitle>
          {t('tasks.detailHeading')}
        </ResourceDetailFallbackTitle>
        <div
          data-testid='task-detail-panel'
          className={`${resourceDetailBodyClassName} flex flex-col gap-3`}
        >
          <ErrorBanner
            message={taskActionErrorMessage(detailQuery.error, t)}
            onRetry={() => void detailQuery.refetch()}
          />
        </div>
      </>
    )
  }

  const task = detailQuery.data
  if (!task) return null

  const userLabel = formatUserLabel(task.user, task.user_id)
  const title = opsIdentity({ userLabel, caseId: task.case_id }) || task.id
  const statusKey = taskStatusLabelKey(task.status)
  const statusLabel = statusKey ? t(statusKey) : task.status
  const showError =
    task.status === 'failed' && Boolean(task.error_code || task.error_message)

  const identifiers: IdentifierItem[] = [
    task.chat_id != null
      ? {
          label: t('tasks.fieldChatId'),
          value: String(task.chat_id),
          copy: true,
        }
      : null,
    task.prompt_id
      ? { label: t('tasks.fieldPromptId'), value: task.prompt_id, copy: true }
      : null,
    task.started_at
      ? {
          label: t('tasks.fieldStartedAt'),
          value: formatDateTime(task.started_at),
        }
      : null,
    task.completed_at
      ? {
          label: t('tasks.fieldCompletedAt'),
          value: formatDateTime(task.completed_at),
        }
      : null,
    {
      label: t('tasks.fieldCreatedAt'),
      value: formatDateTime(task.created_at),
    },
    {
      label: t('tasks.fieldUpdatedAt'),
      value: formatDateTime(task.updated_at),
    },
  ].filter((item): item is IdentifierItem => item != null)

  return (
    <>
      <ResourceDetailLayout
        testId='task-detail-panel'
        title={title}
        status={statusLabel}
        statusClassName={lifecycleBadgeClass(task.status)}
        recordId={task.id}
        error={
          showError ? (
            <Alert variant='destructive'>
              <CircleAlert />
              <AlertTitle>
                {task.error_code || t('tasks.fieldErrorCode')}
              </AlertTitle>
              {task.error_message ? (
                <AlertDescription>{task.error_message}</AlertDescription>
              ) : null}
            </Alert>
          ) : null
        }
        related={
          <>
            {task.user_id ? (
              <RelatedChip
                label={t('tasks.fieldUser')}
                value={userLabel || task.user_id}
                onClick={
                  onOpenRelated
                    ? () => onOpenRelated({ kind: 'user', id: task.user_id! })
                    : undefined
                }
              />
            ) : null}
            {task.session_id ? (
              <RelatedChip
                label={t('tasks.fieldSessionId')}
                value={task.session_id}
                onClick={
                  onOpenRelated
                    ? () =>
                        onOpenRelated({
                          kind: 'session',
                          id: task.session_id,
                        })
                    : undefined
                }
              />
            ) : null}
            {task.channel_id ? (
              <RelatedChip
                label={t('tasks.fieldPlatform')}
                value={task.channel_name || task.channel_id}
                to='/channels/$id'
                params={{ id: task.channel_id }}
              />
            ) : task.channel_name ? (
              <RelatedChip
                label={t('tasks.fieldPlatform')}
                value={task.channel_name}
              />
            ) : null}
            <RelatedChip
              label={t('tasks.fieldCaseId')}
              value={String(task.case_id)}
              to='/cases/$caseId'
              params={{ caseId: String(task.case_id) }}
            />
            {task.edge_id ? (
              <RelatedChip
                label={t('tasks.fieldEdgeId')}
                value={task.edge_id}
                to='/edges/$edgeId'
                params={{ edgeId: task.edge_id }}
              />
            ) : null}
            {task.dispatch_topic ? (
              <RelatedChip
                label={t('tasks.fieldTopic')}
                value={task.dispatch_topic}
                to='/topics/$key'
                params={{ key: task.dispatch_topic }}
              />
            ) : null}
          </>
        }
        identifiers={identifiers}
      />
      {cancellableStatuses.has(task.status) ? (
        <DialogFooter>
          <Button
            type='button'
            variant='destructive'
            size='sm'
            disabled={cancelMut.isPending}
            onClick={() => cancelMut.mutate()}
          >
            {t('tasks.cancel')}
          </Button>
        </DialogFooter>
      ) : null}
    </>
  )
}

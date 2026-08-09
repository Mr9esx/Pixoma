import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { createFileRoute, useNavigate, useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { listTasks } from '@/lib/api/tasks'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { TaskDetailPanel } from '@/features/tasks/detail-panel'
import {
  TaskListPanel,
  type TaskListFilters,
} from '@/features/tasks/list-panel'

export const Route = createFileRoute('/_app/tasks')({
  component: TasksLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function TasksLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { taskId } = useParams({ strict: false }) as { taskId?: string }

  const [filters, setFilters] = useState<TaskListFilters>({
    status: 'all',
    q: '',
  })

  const listParams = {
    status: filters.status === 'all' ? undefined : filters.status,
    q: filters.q.trim() || undefined,
  }

  const listQuery = useQuery({
    queryKey: [...queryKeys.tasks.all, listParams] as const,
    queryFn: () => listTasks(listParams),
  })

  return (
    <div
      className='flex min-h-0 flex-1 flex-col gap-3'
      data-testid='tasks-page'
    >
      <div className='shrink-0'>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('tasks.title')}
        </h1>
        <p className='text-sm text-muted-foreground'>
          {t('tasks.description')}
        </p>
      </div>
      <MasterDetailShell
        hasSelection={Boolean(taskId)}
        onBackToList={() => void navigate({ to: '/tasks' })}
        list={
          <TaskListPanel
            items={listQuery.data ?? []}
            selectedId={taskId}
            filters={filters}
            onFiltersChange={setFilters}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={taskId ? <TaskDetailPanel id={taskId} /> : null}
      />
    </div>
  )
}

import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { Button } from '@/components/ui/button'
import {
  deleteInstance,
  getInstance,
  getInstanceQueue,
  getInstanceSystem,
  listInstanceTasks,
} from '@/lib/api/instances'
import { queryKeys } from '@/lib/api/query-keys'
import { InstanceForm } from './instance-form'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function JsonBlock({ value }: { value: unknown }) {
  return (
    <pre className='bg-muted max-h-64 overflow-auto rounded-md p-3 text-xs whitespace-pre-wrap break-all'>
      {JSON.stringify(value, null, 2)}
    </pre>
  )
}

function ObservationSection({
  title,
  query,
}: {
  title: string
  query: {
    isLoading: boolean
    isError: boolean
    error: unknown
    data: unknown
    refetch: () => void
  }
}) {
  return (
    <section className='space-y-2'>
      <h3 className='text-sm font-medium'>{title}</h3>
      {query.isError ? (
        <ErrorBanner
          message={errorMessage(query.error)}
          onRetry={() => void query.refetch()}
        />
      ) : null}
      {query.isLoading ? <LoadingSkeleton rows={3} /> : null}
      {!query.isLoading && !query.isError ? (
        <JsonBlock value={query.data} />
      ) : null}
    </section>
  )
}

type Props = {
  id: string
}

export function InstanceDetailPanel({ id }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)

  const detailQuery = useQuery({
    queryKey: queryKeys.instances.detail(id),
    queryFn: () => getInstance(id),
  })

  const systemQuery = useQuery({
    queryKey: queryKeys.instances.system(id),
    queryFn: () => getInstanceSystem(id),
  })

  const queueQuery = useQuery({
    queryKey: queryKeys.instances.queue(id),
    queryFn: () => getInstanceQueue(id),
  })

  const tasksQuery = useQuery({
    queryKey: queryKeys.instances.tasks(id),
    queryFn: () => listInstanceTasks(id, { limit: 50 }),
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteInstance(id),
    onSuccess: async () => {
      setConfirmOpen(false)
      await queryClient.invalidateQueries({ queryKey: queryKeys.instances.all })
      toast.success(t('instances.deleteSuccess'))
      void navigate({ to: '/instances' })
    },
    onError: (err) => {
      toast.error(errorMessage(err) ?? t('common.errorGeneric'))
    },
  })

  if (detailQuery.isLoading) {
    return (
      <div data-testid='instance-detail-panel'>
        <LoadingSkeleton rows={6} />
      </div>
    )
  }

  if (detailQuery.isError) {
    return (
      <div data-testid='instance-detail-panel' className='space-y-3'>
        <ErrorBanner
          message={errorMessage(detailQuery.error)}
          onRetry={() => void detailQuery.refetch()}
        />
      </div>
    )
  }

  const instance = detailQuery.data
  if (!instance) return null

  return (
    <div className='space-y-8' data-testid='instance-detail-panel'>
      <div className='space-y-4'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <h2 className='text-lg font-semibold'>{instance.id}</h2>
            <p className='text-muted-foreground text-sm'>
              {t('instances.editHeading')}
            </p>
          </div>
          <Button
            type='button'
            variant='destructive'
            size='sm'
            onClick={() => setConfirmOpen(true)}
          >
            {t('common.delete')}
          </Button>
        </div>
        <InstanceForm key={instance.updated_at} mode='edit' initial={instance} />
      </div>

      <div className='space-y-6 border-t pt-6'>
        <h2 className='text-base font-semibold'>
          {t('instances.observationTitle')}
        </h2>
        <ObservationSection
          title={t('instances.observationSystem')}
          query={systemQuery}
        />
        <ObservationSection
          title={t('instances.observationQueue')}
          query={queueQuery}
        />
        <ObservationSection
          title={t('instances.observationTasks')}
          query={tasksQuery}
        />
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('instances.deleteConfirmTitle')}
        desc={t('instances.deleteConfirmDesc', { id })}
        confirmText={t('common.delete')}
        cancelBtnText={t('common.cancel')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => deleteMutation.mutate()}
      />
    </div>
  )
}

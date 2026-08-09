import { useQuery } from '@tanstack/react-query'
import {
  createFileRoute,
  useNavigate,
  useParams,
} from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { MasterDetailShell } from '@/components/master-detail/master-detail-shell'
import { InstanceDetailPanel } from '@/features/instances/detail-panel'
import { InstanceForm } from '@/features/instances/instance-form'
import { InstanceListPanel } from '@/features/instances/list-panel'
import { listInstances } from '@/lib/api/instances'
import { queryKeys } from '@/lib/api/query-keys'

export const Route = createFileRoute('/_app/instances')({
  component: InstancesLayout,
})

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

function InstancesLayout() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { instanceId } = useParams({ strict: false }) as {
    instanceId?: string
  }

  const listQuery = useQuery({
    queryKey: queryKeys.instances.all,
    queryFn: listInstances,
  })

  return (
    <div className='space-y-3' data-testid='instances-page'>
      <div>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('instances.title')}
        </h1>
        <p className='text-muted-foreground text-sm'>
          {t('instances.description')}
        </p>
      </div>
      <MasterDetailShell
        hasSelection={Boolean(instanceId)}
        onBackToList={() => void navigate({ to: '/instances' })}
        list={
          <InstanceListPanel
            items={listQuery.data ?? []}
            selectedId={instanceId}
            isLoading={listQuery.isLoading}
            isError={listQuery.isError}
            errorMessage={errorMessage(listQuery.error)}
            onRetry={() => void listQuery.refetch()}
          />
        }
        detail={
          instanceId === 'new' ? (
            <div className='space-y-4'>
              <h2 className='text-lg font-semibold'>
                {t('instances.createHeading')}
              </h2>
              <InstanceForm mode='create' />
            </div>
          ) : instanceId ? (
            <InstanceDetailPanel id={instanceId} />
          ) : null
        }
      />
    </div>
  )
}

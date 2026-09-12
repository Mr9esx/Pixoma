import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { TriangleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { Alert, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { isEdgeOnline } from '@/features/task-flow/lib/topic-binding'
import { queueHasSubscribers } from './lib/queue-binding'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

export function Step4Next({ shared }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })

  const offline = useMemo(() => {
    const topic = shared.topicKey
    if (!topic) return false
    const edges = edgesQuery.data ?? []
    const presence = presenceQuery.data ?? []
    if (shared.selectedEdgeId) {
      return !isEdgeOnline(presence, shared.selectedEdgeId)
    }
    if (!queueHasSubscribers(edges, topic)) return true
    return !edges.some(
      (edge) =>
        queueHasSubscribers([edge], topic) && isEdgeOnline(presence, edge.id)
    )
  }, [
    edgesQuery.data,
    presenceQuery.data,
    shared.selectedEdgeId,
    shared.topicKey,
  ])

  return (
    <WizardChrome
      step={4}
      onBack={shared.onExit}
      backLabel={t('quickConfig.leave')}
    >
      <div className='flex flex-col gap-4'>
        <h3 className='text-sm font-semibold'>
          {t('quickConfig.leftoverTitle')}
        </h3>
        <p className='text-sm text-muted-foreground'>
          {t('quickConfig.leftoverHint', { topic: shared.topicKey ?? '' })}
        </p>
        {offline ? (
          <Alert
            variant='warn'
            className='px-3 py-2'
            data-testid='leftover-offline'
          >
            <TriangleAlert aria-hidden='true' />
            <AlertTitle>{t('quickConfig.nodeNotReady')}</AlertTitle>
          </Alert>
        ) : null}
        <Button
          type='button'
          onClick={() => void navigate({ to: '/channels' })}
        >
          {t('quickConfig.goChannels')}
        </Button>
      </div>
    </WizardChrome>
  )
}

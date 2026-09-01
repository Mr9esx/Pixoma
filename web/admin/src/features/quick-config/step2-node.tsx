import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Info, Plus, TriangleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { CreateEdgeWizard } from '@/features/edges/create-edge-wizard'
import { commitQuickCreate } from './lib/commit'
import { nodeStepCanAdvance, queueHasSubscribers } from './lib/queue-binding'
import { clearQuickConfigSession } from './lib/session'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

export function Step2Node({ shared, next, back }: Props) {
  const { t } = useTranslation()
  const [createOpen, setCreateOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [commitError, setCommitError] = useState<string | undefined>()

  const edgesQuery = useQuery({
    queryKey: queryKeys.edges.all,
    queryFn: listEdges,
  })
  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
  })

  const edges = edgesQuery.data ?? []
  const presence = presenceQuery.data ?? []
  const hasSubscribers = shared.topicKey
    ? queueHasSubscribers(edges, shared.topicKey)
    : false
  const canAdvance = nodeStepCanAdvance(hasSubscribers, shared.selectedEdgeId)
  const selectedEdge = edges.find((row) => row.id === shared.selectedEdgeId)
  const selectedOnline = Boolean(
    presence.find(
      (row) =>
        row.id === selectedEdge?.id && row.edge_online && row.comfy_running
    )
  )

  async function handleNext() {
    if (!canAdvance || !shared.caseRecord || !shared.topicKey) return
    setBusy(true)
    setCommitError(undefined)
    try {
      const edge = edges.find((row) => row.id === shared.selectedEdgeId)
      const saved = await commitQuickCreate({
        draft: shared.caseRecord,
        topicKey: shared.topicKey,
        topicDraft: shared.topicDraft,
        selectedEdgeId: shared.selectedEdgeId,
        subscribedTopics: edge?.subscribe_topics ?? [],
      })
      shared.updateCase(saved)
      shared.markCommitted()
      clearQuickConfigSession(window.localStorage)
      next({})
    } catch (err) {
      setCommitError(
        err instanceof Error ? err.message : t('quickConfig.saveFailed')
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <WizardChrome
      step={3}
      onBack={() => back({})}
      onNext={() => void handleNext()}
      nextLabel={t('quickConfig.next')}
      nextDisabled={!canAdvance || busy || !shared.caseRecord}
    >
      <div className='space-y-3'>
        <div className='flex items-center justify-between gap-2'>
          <h3 className='text-sm font-semibold'>
            {t('quickConfig.nodeSelection')}
          </h3>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={() => setCreateOpen(true)}
          >
            <Plus className='size-4' />
            {t('quickConfig.newNode')}
          </Button>
        </div>

        {edgesQuery.isLoading ? null : edges.length === 0 ? (
          <Alert variant='info' className='px-3 py-2'>
            <Info aria-hidden='true' />
            <AlertTitle>{t('quickConfig.noEdgesHint')}</AlertTitle>
          </Alert>
        ) : !canAdvance ? (
          <Alert variant='warn' className='px-3 py-2'>
            <TriangleAlert aria-hidden='true' />
            <AlertTitle>{t('quickConfig.nodeRequired')}</AlertTitle>
          </Alert>
        ) : null}
        {selectedEdge && !selectedOnline ? (
          <Alert variant='warn' className='px-3 py-2'>
            <TriangleAlert aria-hidden='true' />
            <AlertTitle>{t('quickConfig.nodeNotReady')}</AlertTitle>
          </Alert>
        ) : null}
        {commitError ? (
          <Alert variant='destructive' className='px-3 py-2'>
            <TriangleAlert aria-hidden='true' />
            <AlertTitle>{t('quickConfig.saveFailed')}</AlertTitle>
            <AlertDescription>{commitError}</AlertDescription>
          </Alert>
        ) : null}

        {edgesQuery.isLoading ? (
          <LoadingSkeleton rows={2} />
        ) : edges.length === 0 ? null : (
          <ul className='space-y-1'>
            {edges.map((edge) => {
              const online = presence.find(
                (row) =>
                  row.id === edge.id && row.edge_online && row.comfy_running
              )
              return (
                <li key={edge.id}>
                  <button
                    type='button'
                    onClick={() => shared.updateSelectedEdge(edge.id)}
                    className={cn(
                      'flex w-full items-center justify-between rounded-md border border-border px-3 py-2.5 text-left',
                      shared.selectedEdgeId === edge.id &&
                        'border-primary bg-muted/60'
                    )}
                  >
                    <span className='text-sm font-medium'>{edge.name}</span>
                    <span
                      className={cn(
                        'shrink-0 text-xs',
                        online ? 'text-emerald-600' : 'text-muted-foreground'
                      )}
                    >
                      {online
                        ? t('quickConfig.nodeOnline')
                        : t('quickConfig.nodeOffline')}
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className='sm:max-w-[504px]'>
          <DialogHeader>
            <DialogTitle>{t('quickConfig.newNode')}</DialogTitle>
          </DialogHeader>
          <CreateEdgeWizard
            onCancel={() => setCreateOpen(false)}
            onDone={(edge) => {
              shared.updateSelectedEdge(edge.id)
              setCreateOpen(false)
            }}
          />
        </DialogContent>
      </Dialog>
    </WizardChrome>
  )
}

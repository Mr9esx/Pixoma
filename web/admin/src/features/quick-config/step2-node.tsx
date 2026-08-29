import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listEdges, listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { CreateEdgeWizard } from '@/features/edges/create-edge-wizard'
import { cn } from '@/lib/utils'
import type { StepActions, WizardShared } from './types'
import { WizardChrome } from './wizard-chrome'

type Props = StepActions & { shared: WizardShared }

/** Step 2 运行节点：选择已有 Edge 或新建；仅记录 selectedEdgeId，无任何写请求。 */
export function Step2Node({ shared, next, back }: Props) {
  const { t } = useTranslation()
  const [createOpen, setCreateOpen] = useState(false)

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

  return (
    <WizardChrome
      step={2}
      onBack={() => back({})}
      onNext={() => next({})}
      nextLabel={t('quickConfig.next')}
      nextDisabled={!shared.selectedEdgeId}
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

        {edgesQuery.isLoading ? (
          <LoadingSkeleton rows={2} />
        ) : edges.length === 0 ? (
          <p className='text-xs text-muted-foreground'>
            {t('quickConfig.noEdgesHint')}
          </p>
        ) : (
          <ul className='space-y-1'>
            {edges.map((edge) => {
              const online = presence.find(
                (row) => row.id === edge.id && row.edge_online && row.comfy_running
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

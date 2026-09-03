import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { buildLinkGraph, type TopologyKind } from './lib/build-link-graph'
import { LinkGraph } from './link-graph'
import { useTopologySource } from './use-topology-source'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function TopologyOpenButton({
  kind,
  id,
}: {
  kind: TopologyKind
  id: string
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  return (
    <>
      <Button
        type='button'
        variant='outline'
        size='sm'
        data-testid='topology-open'
        onClick={() => setOpen(true)}
      >
        {t('topology.open')}
      </Button>
      <TopologyDialog
        open={open}
        onOpenChange={setOpen}
        kind={kind}
        id={id}
      />
    </>
  )
}

function TopologyDialog({
  open,
  onOpenChange,
  kind,
  id,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  kind: TopologyKind
  id: string
}) {
  const { t } = useTranslation()
  const { source, isLoading, isError, error, refetch } = useTopologySource({
    enabled: open,
  })
  const graph = source
    ? buildLinkGraph(source, { type: 'focus', kind, id })
    : { nodes: [], edges: [] }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='flex h-[min(80vh,720px)] max-w-[90vw] flex-col gap-0 p-0 sm:max-w-[90vw]'>
        <DialogHeader className='border-b px-5 py-4'>
          <DialogTitle>{t('topology.title')}</DialogTitle>
        </DialogHeader>
        <div className='min-h-0 flex-1 p-4'>
          {isLoading ? (
            <LoadingSkeleton rows={8} />
          ) : isError ? (
            <ErrorBanner
              message={errorMessage(error)}
              onRetry={() => refetch()}
            />
          ) : (
            <div className='h-full min-h-[320px]'>
              <LinkGraph graph={graph} />
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

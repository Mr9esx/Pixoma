import { useTranslation } from 'react-i18next'
import { Maximize2, X } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from '@/components/ui/dialog'
import { ErrorBanner } from '@/components/feedback/error-banner'
import { LoadingSkeleton } from '@/components/feedback/loading-skeleton'
import { LinkGraph } from './link-graph'
import { useTopologySource } from './use-topology-source'

function errorMessage(err: unknown): string | undefined {
  return err instanceof Error ? err.message : undefined
}

export function TopologyCard() {
  const { t } = useTranslation()
  const { graphAll, isLoading, isError, error, refetch } = useTopologySource()
  const [open, setOpen] = useState(false)

  return (
    <Card
      data-testid='topology-card'
      className='min-w-0 flex-1 gap-3 py-4'
    >
      <CardHeader className='gap-3 px-4'>
        <CardTitle className='text-base font-semibold'>
          {t('topology.title')}
        </CardTitle>
        <CardAction>
          <Button
            type='button'
            variant='ghost'
            size='icon-sm'
            aria-label={t('topology.fullscreen')}
            onClick={() => setOpen(true)}
          >
            <Maximize2 className='size-4' />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className='grid min-h-0 flex-1 gap-1.5 px-4'>
        <div className='h-[360px]'>
          {isLoading ? (
            <LoadingSkeleton rows={6} />
          ) : isError ? (
            <ErrorBanner
              message={errorMessage(error)}
              onRetry={() => refetch()}
            />
          ) : (
            <LinkGraph graph={graphAll} />
          )}
        </div>
      </CardContent>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          className='h-screen w-screen max-w-none rounded-none p-0 sm:max-w-none'
          showCloseButton={false}
        >
          <DialogTitle className='sr-only'>{t('topology.title')}</DialogTitle>
          <div className='absolute top-4 right-4 z-10'>
            <Button
              type='button'
              variant='outline'
              size='icon'
              aria-label='关闭'
              onClick={() => setOpen(false)}
            >
              <X className='size-4' />
            </Button>
          </div>
          <div className='h-full'>
            <LinkGraph graph={graphAll} />
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  )
}

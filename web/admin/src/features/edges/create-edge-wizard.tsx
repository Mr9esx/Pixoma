import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Check, RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import type { ComfyEdge } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { DialogFooter } from '@/components/ui/dialog'
import { PixomaLoading } from '@/components/feedback/pixoma-loading'
import { DeployCredentials } from './deploy-credentials'
import { EdgeForm } from './edge-form'
import { PresenceTags } from './presence-tags'

type Step = 'form' | 'deploy'

const STEP_ORDER: Step[] = ['form', 'deploy']

type Props = {
  onDone: (edge: ComfyEdge, action: 'view' | 'close') => void
  onCancel: () => void
}

export function CreateEdgeWizard({ onDone, onCancel }: Props) {
  const { t } = useTranslation()
  const [step, setStep] = useState<Step>('form')
  const [edge, setEdge] = useState<ComfyEdge | null>(null)
  const [statusChecked, setStatusChecked] = useState(false)
  const [spinning, setSpinning] = useState(false)

  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
    enabled: step === 'deploy' && edge != null,
    refetchInterval: 3000,
  })
  const presence = presenceQuery.data?.find((row) => row.id === edge?.id)
  const ready =
    presence?.edge_online === true && presence?.comfy_running === true
  const handleCheckStatus = () => {
    if (spinning) return
    setSpinning(true)
    void presenceQuery.refetch()
    // 图标转一圈（与动画时长一致）后再弹出检查结果
    window.setTimeout(() => {
      setSpinning(false)
      setStatusChecked(true)
    }, 600)
  }

  const stepIndex = STEP_ORDER.indexOf(step)
  const steps = [
    { label: t('edges.stepInfo') },
    { label: t('edges.stepDeploy') },
  ]
  return (
    <>
      <ol
        className='flex shrink-0 items-center justify-center gap-2 py-3'
        data-testid='create-edge-steps'
      >
        {steps.map((item, i) => {
          const active = i === stepIndex
          const done = i < stepIndex
          return (
            <li key={item.label} className='flex items-center gap-2'>
              <span
                className={cn(
                  'flex size-6 shrink-0 items-center justify-center rounded-full text-xs font-medium',
                  active && 'bg-primary text-primary-foreground',
                  !active && done && 'bg-primary/20 text-primary',
                  !active && !done && 'bg-muted text-muted-foreground'
                )}
              >
                {done ? <Check className='size-3.5' aria-hidden /> : i + 1}
              </span>
              <span
                className={cn(
                  'truncate text-xs',
                  active
                    ? 'font-medium text-foreground'
                    : 'text-muted-foreground'
                )}
              >
                {item.label}
              </span>
              {i < steps.length - 1 ? (
                <span
                  className={cn(
                    'h-px w-6 shrink-0',
                    done ? 'bg-primary/40' : 'bg-border'
                  )}
                  aria-hidden
                />
              ) : null}
            </li>
          )
        })}
      </ol>
      {step === 'form' ? (
        <EdgeForm
          mode='create'
          layout='dialog'
          onCancel={onCancel}
          onSaved={(created) => {
            setEdge(created)
            setStep('deploy')
          }}
        />
      ) : step === 'deploy' && edge ? (
        <>
          <div
            className='flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-1'
            data-testid='edge-deploy-step'
          >
            <DeployCredentials edge={edge} showToken={false} />
            <div className='flex flex-wrap items-center gap-2'>
              <Button
                type='button'
                variant='outline'
                onClick={handleCheckStatus}
              >
                {spinning ? (
                  <PixomaLoading />
                ) : (
                  <RefreshCw className='size-4' />
                )}
                {t('edges.checkNodeStatus')}
              </Button>
              <PresenceTags
                edgeOnline={presence?.edge_online === true}
                comfyRunning={presence?.comfy_running === true}
              />
            </div>
            {statusChecked && !ready ? (
              <Alert variant='warn'>
                <AlertTriangle className='size-4' aria-hidden />
                <AlertDescription>
                  {t('edges.checkStatusFailed')}
                </AlertDescription>
              </Alert>
            ) : null}
          </div>
          <DialogFooter className='shrink-0'>
            <Button
              type='button'
              variant='ghost'
              onClick={() => edge && onDone(edge, 'close')}
            >
              {t('edges.deploySkip')}
            </Button>
            <Button
              type='button'
              disabled={!ready}
              onClick={() => edge && onDone(edge, 'view')}
            >
              {t('edges.deployContinue')}
            </Button>
          </DialogFooter>
        </>
      ) : null}
    </>
  )
}

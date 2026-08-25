import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Check, CheckCircle2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { listPresence } from '@/lib/api/edges'
import { queryKeys } from '@/lib/api/query-keys'
import type { ComfyEdge } from '@/lib/api/types'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { DeployCredentials } from './deploy-credentials'
import { EdgeForm } from './edge-form'
import { PresenceTags } from './presence-tags'

type Step = 'form' | 'deploy' | 'done'

const STEP_ORDER: Step[] = ['form', 'deploy', 'done']

type Props = {
  onDone: (edge: ComfyEdge, action: 'view' | 'close') => void
}

export function CreateEdgeWizard({ onDone }: Props) {
  const { t } = useTranslation()
  const [step, setStep] = useState<Step>('form')
  const [edge, setEdge] = useState<ComfyEdge | null>(null)

  const presenceQuery = useQuery({
    queryKey: queryKeys.edges.presence,
    queryFn: listPresence,
    enabled: step === 'deploy' && edge != null,
    refetchInterval: 3000,
  })
  const presence = presenceQuery.data?.find((row) => row.id === edge?.id)
  const ready =
    presence?.edge_online === true && presence?.comfy_running === true

  const stepIndex = STEP_ORDER.indexOf(step)
  const steps = [
    { label: t('edges.stepInfo') },
    { label: t('edges.stepDeploy') },
    { label: t('edges.stepDone') },
  ]
  const title =
    step === 'form'
      ? t('edges.createNode')
      : step === 'deploy'
        ? t('edges.deployHeading')
        : t('edges.createDoneHeading')

  return (
    <>
      <div className='shrink-0'>
        <h2 className='text-lg font-semibold tracking-tight'>{title}</h2>
      </div>
      <ol
        className='flex shrink-0 items-center justify-center gap-2 py-6'
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
          onSaved={(created) => {
            setEdge(created)
            setStep('deploy')
          }}
        />
      ) : step === 'deploy' && edge ? (
        <div
          className='flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-1'
          data-testid='edge-deploy-step'
        >
          <DeployCredentials edge={edge} />
          <div className='flex flex-wrap items-center gap-2'>
            <PresenceTags
              edgeOnline={presence?.edge_online === true}
              comfyRunning={presence?.comfy_running === true}
            />
          </div>
          <p className='text-xs text-muted-foreground'>
            {t('edges.deployWaitHint')}
          </p>
          <div className='flex shrink-0 items-center justify-end gap-2'>
            <Button
              type='button'
              variant='ghost'
              onClick={() => setStep('done')}
            >
              {t('edges.deploySkip')}
            </Button>
            <Button
              type='button'
              disabled={!ready}
              onClick={() => setStep('done')}
            >
              {t('edges.deployContinue')}
            </Button>
          </div>
        </div>
      ) : (
        <>
          <div
            className='flex min-h-0 flex-1 flex-col items-center gap-4 overflow-y-auto py-6 text-center'
            data-testid='edge-create-result'
          >
            <CheckCircle2 className='size-10 text-primary' aria-hidden />
            <div className='flex flex-col gap-1'>
              <p className='font-medium'>{edge?.name}</p>
              <p className='text-sm text-muted-foreground'>
                {t('edges.createDoneDesc')}
              </p>
            </div>
          </div>
          <div className='flex shrink-0 items-center justify-end gap-2'>
            <Button
              type='button'
              variant='outline'
              onClick={() => edge && onDone(edge, 'close')}
            >
              {t('edges.createDoneClose')}
            </Button>
            <Button type='button' onClick={() => edge && onDone(edge, 'view')}>
              {t('edges.createDoneView')}
            </Button>
          </div>
        </>
      )}
    </>
  )
}

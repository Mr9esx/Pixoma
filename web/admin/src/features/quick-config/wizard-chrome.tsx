import { Fragment, type ReactNode } from 'react'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

type Props = {
  step: 1 | 2 | 3 | 4
  onBack?: () => void
  onNext?: () => void
  nextLabel?: string
  nextDisabled?: boolean
  backLabel?: string
  hideFooter?: boolean
  children: ReactNode
}

export function WizardChrome({
  step,
  onBack,
  onNext,
  nextLabel,
  nextDisabled,
  backLabel,
  hideFooter,
  children,
}: Props) {
  const { t } = useTranslation()
  const STEP_LABELS = [
    t('quickConfig.workflowConfig'),
    t('quickConfig.queueStep'),
    t('quickConfig.nodeSelection'),
    t('quickConfig.leftoverTitle'),
  ]
  return (
    <div
      className='flex h-full min-h-0 flex-col gap-4'
      data-testid='quick-config-chrome'
    >
      <div className='flex shrink-0 items-center justify-center rounded-xl border border-border bg-background px-4 py-3'>
        <div className='flex min-w-0 flex-wrap items-center justify-center gap-x-3 gap-y-2'>
          {[1, 2, 3, 4].map((i) => (
            <Fragment key={i}>
              <div className='flex items-center gap-1.5'>
                <span
                  className={cn(
                    'grid size-5 place-items-center rounded-full border text-[11px] font-semibold',
                    i < step
                      ? 'border-emerald-600/50 text-emerald-600'
                      : i === step
                        ? 'border-foreground bg-foreground text-background'
                        : 'border-border text-muted-foreground'
                  )}
                >
                  {i === 4 ? '✓' : i}
                </span>
                <span
                  className={cn(
                    'text-xs',
                    i === step
                      ? 'font-semibold text-foreground'
                      : 'text-muted-foreground'
                  )}
                >
                  {STEP_LABELS[i - 1]}
                </span>
              </div>
              {i < 4 ? (
                <span
                  aria-hidden='true'
                  className={cn(
                    'h-px min-w-4 flex-1',
                    i < step ? 'bg-emerald-600/50' : 'bg-border'
                  )}
                />
              ) : null}
            </Fragment>
          ))}
        </div>
      </div>

      <div className='min-h-0 flex-1 overflow-auto rounded-xl border border-border bg-background p-5'>
        {children}
      </div>

      {hideFooter ? null : (
        <div className='flex shrink-0 items-center justify-between rounded-xl border border-border bg-background px-4 py-3'>
          <Button
            type='button'
            variant='outline'
            onClick={onBack}
            disabled={!onBack}
          >
            <ArrowLeft className='size-4' />
            {backLabel ?? t('quickConfig.back')}
          </Button>
          {onNext ? (
            <Button type='button' onClick={onNext} disabled={nextDisabled}>
              {nextLabel ?? t('quickConfig.nextSave')}
              <ArrowRight className='size-4' />
            </Button>
          ) : null}
        </div>
      )}
    </div>
  )
}

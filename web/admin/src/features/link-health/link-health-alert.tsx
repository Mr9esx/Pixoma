import { AlertCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import type { EntityHealth } from './types'

type LinkHealthAlertProps = {
  name: string
  health: EntityHealth
  /** 页面内锚点，指向下方健康区块，例如 '#link-health-section'。 */
  anchorTo: string
}

export function LinkHealthAlert({
  health,
  anchorTo,
}: LinkHealthAlertProps) {
  const { t } = useTranslation()
  if (health.state === 'ok') return null
  const n = health.breakpoints.length
  if (n === 0) return null
  return (
    <Alert
      variant='warn'
      data-testid='link-health-alert'
      className='flex flex-wrap items-center justify-between gap-3'
    >
      <div className='flex min-w-0 flex-wrap items-center gap-2'>
        <AlertCircle className='size-4 shrink-0' aria-hidden='true' />
        <AlertTitle className='text-sm font-medium'>
          {t(health.breakpoints[0].key, health.breakpoints[0].params)}
        </AlertTitle>
        <AlertDescription className='text-sm'>
          {t('linkHealth.alertSummary', { n })}
        </AlertDescription>
      </div>
      <a
        href={anchorTo}
        className='text-sm font-medium underline underline-offset-2'
      >
        {t('linkHealth.alertViewDetails')} ↓
      </a>
    </Alert>
  )
}

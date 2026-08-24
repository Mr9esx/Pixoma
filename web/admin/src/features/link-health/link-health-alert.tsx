import { useTranslation } from 'react-i18next'
import { AlertCircle } from 'lucide-react'
import type { EntityHealth } from './lib/references'

export type LinkHealthAlertProps = {
  name: string
  health: EntityHealth
  /** 页面内锚点，指向下方健康区块，例如 '#link-health-section'。 */
  anchorTo: string
}

export function LinkHealthAlert({ name, health, anchorTo }: LinkHealthAlertProps) {
  const { t } = useTranslation()
  if (health.state === 'ok') return null
  const n = health.breakpoints.length
  return (
    <div
      role='alert'
      data-testid='link-health-alert'
      className='flex flex-wrap items-center justify-between gap-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-300'
    >
      <div className='flex min-w-0 flex-wrap items-center gap-2'>
        <AlertCircle className='size-4 shrink-0' aria-hidden='true' />
        <span className='font-medium'>{t('linkHealth.alertTitle', { name })}</span>
        <span className='text-amber-700/70 dark:text-amber-300/70'>
          {t('linkHealth.alertSummary', { n })}
        </span>
      </div>
      <a href={anchorTo} className='font-medium underline underline-offset-2'>
        {t('linkHealth.alertViewDetails')} ↓
      </a>
    </div>
  )
}

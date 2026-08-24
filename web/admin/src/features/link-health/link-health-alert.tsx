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
      className='flex flex-wrap items-center gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-300'
    >
      <AlertCircle className='size-4 shrink-0' aria-hidden='true' />
      <span className='font-medium'>{t('linkHealth.alertTitle', { name })}</span>
      <span className='text-muted-foreground'>
        {t('linkHealth.alertSummary', { n })}
      </span>
      <a href={anchorTo} className='ml-auto font-medium underline underline-offset-2'>
        {t('linkHealth.alertViewDetails')} ↓
      </a>
    </div>
  )
}

import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Reveal } from '@/components/ui/reveal'
import { cn } from '@/lib/utils'

type Props = {
  message?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ message, action, className }: Props) {
  const { t } = useTranslation()
  return (
    <Reveal
      className={cn(
        'flex flex-col items-center justify-center gap-3 py-12 text-center',
        className
      )}
    >
      <p className='text-muted-foreground'>{message ?? t('common.empty')}</p>
      {action}
    </Reveal>
  )
}

import { useTranslation } from 'react-i18next'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Reveal } from '@/components/ui/reveal'
import { cn } from '@/lib/utils'

type Props = {
  message?: string
  onRetry?: () => void
  className?: string
}

export function ErrorBanner({ message, onRetry, className }: Props) {
  const { t } = useTranslation()
  return (
    <Reveal className={cn(className)}>
      <Alert
        variant='destructive'
        className='flex flex-wrap items-center justify-between gap-3'
      >
        <p className='min-w-0 text-sm'>{message ?? t('common.errorGeneric')}</p>
        {onRetry ? (
          <Button type='button' variant='outline' size='sm' onClick={onRetry}>
            {t('common.retry')}
          </Button>
        ) : null}
      </Alert>
    </Reveal>
  )
}

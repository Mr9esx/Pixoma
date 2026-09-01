import { useTranslation } from 'react-i18next'
import { DotPatternBackground } from '@/components/feedback/dot-pattern-background'
import { PixomaLoading } from '@/components/feedback/pixoma-loading'

export function LoadingPage() {
  const { t } = useTranslation()

  return (
    <div
      className='relative grid min-h-svh place-items-center bg-background'
      role='status'
      aria-label={t('common.loading')}
    >
      <DotPatternBackground />
      <PixomaLoading className='relative' />
    </div>
  )
}

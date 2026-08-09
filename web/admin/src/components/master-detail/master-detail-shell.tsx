import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

type Props = {
  list: ReactNode
  detail: ReactNode
  hasSelection: boolean
  onBackToList?: () => void
  className?: string
}

export function MasterDetailShell({
  list,
  detail,
  hasSelection,
  onBackToList,
  className,
}: Props) {
  const { t } = useTranslation()
  return (
    <div
      className={cn(
        'grid h-[calc(100vh-5rem)] gap-0 overflow-hidden rounded-md border',
        'md:grid-cols-[minmax(280px,360px)_1fr]',
        className
      )}
    >
      <aside
        className={cn(
          'min-h-0 overflow-auto border-r',
          hasSelection ? 'hidden md:block' : 'block'
        )}
      >
        {list}
      </aside>
      <section
        className={cn(
          'min-h-0 overflow-auto p-4',
          !hasSelection ? 'hidden md:block' : 'block'
        )}
      >
        {hasSelection ? (
          <>
            {onBackToList ? (
              <button
                type='button'
                className='mb-3 text-sm underline md:hidden'
                onClick={onBackToList}
              >
                {t('common.backToList', { defaultValue: '返回列表' })}
              </button>
            ) : null}
            {detail}
          </>
        ) : (
          <p className='text-muted-foreground'>{t('common.selectItem')}</p>
        )}
      </section>
    </div>
  )
}

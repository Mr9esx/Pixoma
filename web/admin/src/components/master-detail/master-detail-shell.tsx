import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'

type Props = {
  list: ReactNode
  detail: ReactNode
  hasSelection: boolean
  onBackToList?: () => void
  emptyDetail?: ReactNode
  className?: string
  detailClassName?: string
}

export function MasterDetailShell({
  list,
  detail,
  hasSelection,
  onBackToList,
  emptyDetail,
  className,
  detailClassName,
}: Props) {
  const { t } = useTranslation()
  return (
    <div
      className={cn(
        'grid min-h-0 flex-1 gap-0 overflow-hidden rounded-md border',
        'md:grid-cols-[minmax(280px,360px)_minmax(0,1fr)]',
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
          'min-h-0',
          detailClassName ?? 'overflow-auto p-4',
          !hasSelection && 'hidden md:block'
        )}
      >
        {hasSelection ? (
          <>
            {onBackToList ? (
              <button
                type='button'
                className='mb-3 px-4 pt-3 text-sm underline md:hidden'
                onClick={onBackToList}
              >
                {t('common.backToList', { defaultValue: '返回列表' })}
              </button>
            ) : null}
            {detail}
          </>
        ) : (
          (emptyDetail ?? (
            <p className='p-4 text-muted-foreground'>
              {t('common.selectItem')}
            </p>
          ))
        )}
      </section>
    </div>
  )
}

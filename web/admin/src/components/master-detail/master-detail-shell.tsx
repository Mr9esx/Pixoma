import { isValidElement, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { useScrollFocus } from '@/lib/scroll-focus'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Reveal } from '@/components/ui/reveal'

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
  useScrollFocus()
  const detailKey = isValidElement(detail)
    ? typeof detail.key === 'string'
      ? detail.key
      : 'detail'
    : 'detail'

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
          !hasSelection && 'hidden md:flex'
        )}
      >
        {hasSelection ? (
          <>
            {onBackToList ? (
              <Button
                type='button'
                variant='link'
                size='sm'
                className='mb-3 justify-start px-4 pt-3 md:hidden'
                onClick={onBackToList}
              >
                {t('common.backToList', { defaultValue: '返回列表' })}
              </Button>
            ) : null}
            <Reveal key={detailKey} className='flex min-h-0 flex-1 flex-col'>
              {detail}
            </Reveal>
          </>
        ) : (
          <Reveal className='flex min-h-0 flex-1 flex-col'>
            {emptyDetail ?? null}
          </Reveal>
        )}
      </section>
    </div>
  )
}

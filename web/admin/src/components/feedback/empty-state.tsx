import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
} from '@/components/ui/empty'
import { Reveal } from '@/components/ui/reveal'

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
      <Empty className='gap-3 p-0 md:p-0'>
        <EmptyHeader className='max-w-none gap-0'>
          <EmptyDescription>{message ?? t('common.empty')}</EmptyDescription>
        </EmptyHeader>
        {action ? (
          <EmptyContent className='w-auto flex-row justify-center gap-2'>
            {action}
          </EmptyContent>
        ) : null}
      </Empty>
    </Reveal>
  )
}

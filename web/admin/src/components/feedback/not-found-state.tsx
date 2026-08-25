import type { ReactNode } from 'react'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'

type Props = {
  icon: ReactNode
  title: string
  description: string
  actions?: ReactNode
}

export function NotFoundState({ icon, title, description, actions }: Props) {
  return (
    <div className='flex min-h-0 flex-1 flex-col'>
      <Empty>
        <EmptyHeader className='max-w-none'>
          <EmptyMedia variant='icon'>{icon}</EmptyMedia>
          <EmptyTitle className='text-sm font-medium'>{title}</EmptyTitle>
          <EmptyDescription>{description}</EmptyDescription>
        </EmptyHeader>
        {actions ? (
          <EmptyContent className='flex-row justify-center gap-2'>
            {actions}
          </EmptyContent>
        ) : null}
      </Empty>
    </div>
  )
}

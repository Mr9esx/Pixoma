import type { ReactNode } from 'react'
import { kit } from '@/features/edges/kit-classes'

export function MetaChip({
  icon,
  label,
  value,
  divider = false,
}: {
  icon: ReactNode
  label: string
  value?: string
  divider?: boolean
}) {
  return (
    <div className={kit.metaChip}>
      <div className='flex min-w-0 items-center gap-2 text-muted-foreground'>
        {icon}
        <span className='shrink-0'>{label}</span>
        <span className='min-w-0 truncate font-medium text-foreground'>
          {value || '—'}
        </span>
      </div>
      {divider ? (
        <div
          data-orientation='vertical'
          role='none'
          className={kit.metaChipDivider}
        />
      ) : null}
    </div>
  )
}

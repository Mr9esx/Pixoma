import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { resourceDetailBodyClassName } from '@/features/resource-modal'
import { CopyId } from './copy-id'

const chipClassName =
  'inline-flex h-8 max-w-full min-w-0 items-center gap-1.5 rounded-md border border-border px-2.5 text-xs'

type RelatedChipProps = {
  label: string
  value: string
  onClick?: () => void
  to?: '/channels/$id' | '/cases/$caseId' | '/edges/$edgeId' | '/topics/$key'
  params?: { id?: string; caseId?: string; edgeId?: string; key?: string }
}

export function RelatedChip({
  label,
  value,
  onClick,
  to,
  params,
}: RelatedChipProps) {
  const inner = (
    <>
      <span className='shrink-0 text-muted-foreground'>{label}</span>
      <span className='min-w-0 truncate font-medium'>{value}</span>
    </>
  )
  const clickableClass = cn(chipClassName, 'hover:bg-accent')
  if (to === '/channels/$id' && params?.id) {
    return (
      <Link to={to} params={{ id: params.id }} className={clickableClass}>
        {inner}
      </Link>
    )
  }
  if (to === '/cases/$caseId' && params?.caseId) {
    return (
      <Link
        to={to}
        params={{ caseId: params.caseId }}
        className={clickableClass}
      >
        {inner}
      </Link>
    )
  }
  if (to === '/edges/$edgeId' && params?.edgeId) {
    return (
      <Link
        to={to}
        params={{ edgeId: params.edgeId }}
        className={clickableClass}
      >
        {inner}
      </Link>
    )
  }
  if (to === '/topics/$key' && params?.key) {
    return (
      <Link to={to} params={{ key: params.key }} className={clickableClass}>
        {inner}
      </Link>
    )
  }
  if (onClick) {
    return (
      <button type='button' onClick={onClick} className={clickableClass}>
        {inner}
      </button>
    )
  }
  return <span className={chipClassName}>{inner}</span>
}

export function ResourceDetailFallbackTitle({
  children,
}: {
  children: ReactNode
}) {
  return (
    <DialogHeader className='pr-8'>
      <DialogTitle>{children}</DialogTitle>
    </DialogHeader>
  )
}

export type IdentifierItem = {
  label: string
  value: string
  copy?: boolean
}

type ResourceDetailLayoutProps = {
  title: string
  status: string
  statusClassName?: string
  recordId: string
  error?: ReactNode
  related?: ReactNode
  identifiers: IdentifierItem[]
  extra?: ReactNode
  testId?: string
}

export function ResourceDetailLayout({
  title,
  status,
  statusClassName,
  recordId,
  error,
  related,
  identifiers,
  extra,
  testId,
}: ResourceDetailLayoutProps) {
  const { t } = useTranslation()

  return (
    <>
      <DialogHeader className='shrink-0 pr-8'>
        <div className='flex min-w-0 items-center gap-2'>
          <DialogTitle className='min-w-0 truncate'>{title}</DialogTitle>
          <Badge variant='outline' className={cn('shrink-0', statusClassName)}>
            {status}
          </Badge>
        </div>
        <CopyId value={recordId} />
      </DialogHeader>

      <div
        data-testid={testId}
        className={cn(resourceDetailBodyClassName, 'flex flex-col gap-4')}
      >
        {error}

        {related ? (
          <section className='flex flex-col gap-2'>
            <h3 className='text-xs font-medium text-muted-foreground'>
              {t('operations.related')}
            </h3>
            <div className='flex flex-wrap gap-2'>{related}</div>
          </section>
        ) : null}

        {identifiers.length > 0 ? (
          <section className='flex flex-col gap-2'>
            <h3 className='text-xs font-medium text-muted-foreground'>
              {t('operations.ids')}
            </h3>
            <dl className='grid grid-cols-1 gap-x-4 gap-y-3 sm:grid-cols-2'>
              {identifiers.map((item) => (
                <div key={item.label} className='flex min-w-0 flex-col gap-1'>
                  <dt className='text-xs font-medium text-muted-foreground'>
                    {item.label}
                  </dt>
                  <dd className='min-w-0 text-sm break-all'>
                    {item.copy ? (
                      <CopyId value={item.value} />
                    ) : (
                      <span className='tabular-nums'>{item.value || '—'}</span>
                    )}
                  </dd>
                </div>
              ))}
            </dl>
          </section>
        ) : null}

        {extra}
      </div>
    </>
  )
}

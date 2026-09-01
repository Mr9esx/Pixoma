import type { ReactNode } from 'react'

type DetailFieldProps = {
  label: ReactNode
  value: ReactNode
}

export function DetailField({ label, value }: DetailFieldProps) {
  return (
    <>
      <dt className='text-xs font-medium text-muted-foreground'>{label}</dt>
      <dd className='text-sm break-all'>{value || '—'}</dd>
    </>
  )
}

import { cn } from '@/lib/utils'

type Props = {
  problems: number
  label: string
}

export function StatusDot({ problems, label }: Props) {
  const tone = problems === 0 ? 'ok' : 'warn'

  return (
    <span
      role='img'
      aria-label={label}
      title={label}
      data-health={tone}
      className={cn(
        'size-2 shrink-0 rounded-full',
        problems === 0 ? 'bg-success' : 'bg-warning'
      )}
    />
  )
}

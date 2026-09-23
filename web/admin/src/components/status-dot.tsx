import { cn } from '@/lib/utils'
import { LoaderCircle } from 'lucide-react'

type Props = {
  problems?: number
  label: string
  state?: 'ok' | 'warn' | 'active'
  pulse?: boolean
}

export function StatusDot({ problems = 0, label, state, pulse }: Props) {
  const tone = state ?? (problems === 0 ? 'ok' : 'warn')
  const active = tone === 'active'

  return (
    <span
      role='img'
      aria-label={label}
      title={label}
      data-health={tone}
      data-state={tone}
      className={cn(
        'inline-flex shrink-0 items-center justify-center',
        active ? 'size-3' : 'size-2 rounded-full',
        !active && (tone === 'ok' ? 'bg-success' : 'bg-warning'),
        pulse && !active && 'animate-pulse'
      )}
    >
      {active ? (
        <LoaderCircle
          aria-hidden='true'
          className='size-3 animate-spin text-info'
        />
      ) : null}
    </span>
  )
}

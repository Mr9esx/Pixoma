import type { ComponentProps } from 'react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

export type PillTone = 'success' | 'warning' | 'error' | 'neutral'

type PillProps = ComponentProps<typeof Badge> & {
  dot?: PillTone
  pulse?: boolean
}

const pillDotClass: Record<PillTone, string> = {
  success: 'bg-success',
  warning: 'bg-warning',
  error: 'bg-destructive',
  neutral: 'bg-muted-foreground',
}

function PillDot({ tone, pulse }: { tone: PillTone; pulse: boolean }) {
  return (
    <span aria-hidden className='relative inline-flex size-1.5 shrink-0'>
      {pulse ? (
        <span
          className={cn(
            'absolute inset-0 animate-ping rounded-full opacity-60 [animation-duration:1.6s] motion-reduce:animate-none',
            pillDotClass[tone]
          )}
        />
      ) : null}
      <span
        className={cn(
          'relative inline-flex size-1.5 rounded-full',
          pillDotClass[tone]
        )}
      />
    </span>
  )
}

export const Pill = ({
  variant = 'secondary',
  dot,
  pulse = dot !== undefined && dot !== 'neutral',
  className,
  children,
  ...props
}: PillProps) => (
  <Badge
    className={cn('gap-2 rounded-full px-3 py-1.5 font-normal', className)}
    variant={variant}
    {...props}
  >
    {dot ? <PillDot tone={dot} pulse={pulse} /> : null}
    {children}
  </Badge>
)

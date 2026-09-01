import { cn } from '@/lib/utils'

type DotPatternBackgroundProps = {
  className?: string
}

export function DotPatternBackground({ className }: DotPatternBackgroundProps) {
  return (
    <div
      aria-hidden
      className={cn('pointer-events-none absolute inset-0', className)}
      style={{
        backgroundImage:
          'radial-gradient(circle, color-mix(in oklch, var(--muted-foreground) 25%, transparent) 1px, transparent 1px)',
        backgroundSize: '16px 16px',
      }}
    />
  )
}

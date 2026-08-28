import { useReducedMotion } from 'motion/react'
import { cn } from '@/lib/utils'

export function GradientBackground({ className }: { className?: string }) {
  const reduced = useReducedMotion()

  return (
    <div aria-hidden className={cn('pointer-events-none absolute inset-0 overflow-hidden', className)}>
      <div
        className={cn(
          'absolute -top-24 left-1/2 h-96 w-96 -translate-x-1/2 rounded-full bg-primary/20 blur-3xl',
          !reduced && 'animate-pulse',
        )}
      />
      <div
        className={cn(
          'absolute top-24 right-0 h-72 w-72 rounded-full bg-secondary/40 blur-3xl',
          !reduced && 'animate-pulse',
        )}
      />
    </div>
  )
}

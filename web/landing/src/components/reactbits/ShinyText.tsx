import { useReducedMotion } from 'motion/react'
import { cn } from '@/lib/utils'

type ShinyTextProps = {
  children: string
  className?: string
}

export function ShinyText({ children, className }: ShinyTextProps) {
  const reduced = useReducedMotion()

  return (
    <span
      className={cn(
        'bg-gradient-to-r from-foreground via-primary to-foreground bg-clip-text text-transparent',
        !reduced && 'animate-shiny',
        className,
      )}
    >
      {children}
    </span>
  )
}

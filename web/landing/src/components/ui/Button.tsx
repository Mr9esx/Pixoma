import { type ComponentPropsWithoutRef } from 'react'
import { cn } from '@/lib/utils'

type ButtonProps = ComponentPropsWithoutRef<'a'> & {
  variant?: 'primary' | 'outline'
}

export function Button({ variant = 'primary', className, ...props }: ButtonProps) {
  return (
    <a
      className={cn(
        'inline-flex items-center justify-center gap-2 rounded-lg px-5 py-2.5 text-sm font-medium transition-colors',
        variant === 'primary' && 'bg-primary text-primary-foreground hover:opacity-90',
        variant === 'outline' && 'border border-border bg-card text-foreground hover:bg-muted',
        className,
      )}
      {...props}
    />
  )
}

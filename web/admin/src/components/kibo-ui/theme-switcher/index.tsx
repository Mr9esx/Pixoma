'use client'

import { Moon, Sun } from 'lucide-react'
import { motion } from 'motion/react'
import { cn } from '@/lib/utils'

type ThemeSwitcherProps = {
  value?: 'light' | 'dark'
  onChange?: (theme: 'light' | 'dark') => void
  className?: string
}

export const ThemeSwitcher = ({
  value,
  onChange,
  className,
}: ThemeSwitcherProps) => {
  const isDark = value === 'dark'

  return (
    <button
      aria-checked={isDark}
      aria-label='Toggle theme'
      className={cn(
        'relative isolate flex h-6 rounded-full bg-background p-0.5 ring-1 ring-border ring-inset',
        className
      )}
      onClick={() => onChange?.(isDark ? 'light' : 'dark')}
      role='switch'
      type='button'
    >
      <motion.span
        animate={{ x: isDark ? 20 : 0 }}
        className='absolute inset-y-0.5 left-0.5 w-5 rounded-full bg-secondary'
        transition={{ type: 'spring', duration: 0.5 }}
      />
      <span className='relative z-10 flex h-5 w-5 items-center justify-center'>
        <Sun
          className={cn(
            'size-3.5',
            isDark ? 'text-muted-foreground' : 'text-foreground'
          )}
        />
      </span>
      <span className='relative z-10 flex h-5 w-5 items-center justify-center'>
        <Moon
          className={cn(
            'size-3.5',
            isDark ? 'text-foreground' : 'text-muted-foreground'
          )}
        />
      </span>
    </button>
  )
}

'use client'

import { memo, type CSSProperties } from 'react'
import { motion, useReducedMotion } from 'motion/react'
import { cn } from '@/lib/utils'

export type TextShimmerProps = {
  children: string
  className?: string
  duration?: number
  spread?: number
}

export const Shimmer = memo(function Shimmer({
  children,
  className,
  duration = 2,
  spread = 2,
}: TextShimmerProps) {
  const reducedMotion = useReducedMotion()

  return (
    <motion.p
      animate={{
        backgroundPosition: reducedMotion ? '100% center' : '0% center',
      }}
      className={cn(
        'relative inline-block bg-[length:250%_100%,auto] bg-clip-text text-transparent',
        '[background-repeat:no-repeat,padding-box] [--bg:linear-gradient(90deg,transparent_calc(50%-var(--spread)),var(--color-background),transparent_calc(50%+var(--spread)))]',
        className
      )}
      initial={{ backgroundPosition: '100% center' }}
      style={
        {
          '--spread': `${children.length * spread}px`,
          backgroundImage:
            'var(--bg), linear-gradient(var(--color-muted-foreground), var(--color-muted-foreground))',
        } as CSSProperties
      }
      transition={
        reducedMotion
          ? { duration: 0 }
          : {
              duration,
              ease: 'linear',
              repeat: Number.POSITIVE_INFINITY,
            }
      }
    >
      {children}
    </motion.p>
  )
})

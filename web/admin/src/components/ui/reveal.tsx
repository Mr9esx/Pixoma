import type { ComponentProps } from 'react'
import { motion, useReducedMotion } from 'motion/react'
import { cn } from '@/lib/utils'

type RevealProps = ComponentProps<typeof motion.div> & {
  delay?: number
  as?: 'div' | 'section'
}

/**
 * 内容出现时的统一淡入淡出（纯 opacity，不做位移）。
 * 用于骨架屏→内容、空态/错误态、详情面板等「硬切换」场景。
 * 严格尊重 prefers-reduced-motion：命中时直接渲染普通 div，零动画。
 */
export function Reveal({
  className,
  delay = 0,
  as = 'div',
  children,
  ...props
}: RevealProps) {
  const reduceMotion = useReducedMotion()
  const MotionEl = as === 'section' ? motion.section : motion.div

  if (reduceMotion) {
    return (
      <MotionEl
        className={cn(className)}
        initial={false}
        animate={false}
        {...props}
      >
        {children}
      </MotionEl>
    )
  }

  return (
    <MotionEl
      className={cn(className)}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.16, ease: 'easeOut', delay }}
      {...props}
    >
      {children}
    </MotionEl>
  )
}

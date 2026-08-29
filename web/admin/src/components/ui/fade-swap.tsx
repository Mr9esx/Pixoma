import type { ReactNode } from 'react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'

type FadeSwapProps = {
  /** 作为 key，当值变化时触发淡出/淡入。 */
  value: string | number
  children?: ReactNode
  className?: string
}

/**
 * 文本类内容的淡交换：state 文本（如 running→succeeded）切换时
 * 做 120ms 淡出/淡入，传递「正在变化」的信息。
 * 首次挂载不播放动画；尊重 prefers-reduced-motion。
 */
export function FadeSwap({ value, children, className }: FadeSwapProps) {
  const reduceMotion = useReducedMotion()

  if (reduceMotion) {
    return <span className={className}>{children}</span>
  }

  return (
    <AnimatePresence mode='wait' initial={false}>
      <motion.span
        key={value}
        className={className}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.12, ease: 'easeOut' }}
      >
        {children}
      </motion.span>
    </AnimatePresence>
  )
}

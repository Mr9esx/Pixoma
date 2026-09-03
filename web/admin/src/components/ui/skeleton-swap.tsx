import type { ReactNode } from 'react'
import { motion, useReducedMotion } from 'motion/react'
import { cn } from '@/lib/utils'

type SkeletonSwapProps = {
  /** 加载中显示 skeleton；切到 false 后做交叉淡入/淡出到 children。 */
  loading: boolean
  skeleton: ReactNode
  children: ReactNode
  /**
   * 单段淡入/淡出的秒数。skeleton 淡出与 children 淡入并行触发，
   * 因此整个过渡 ≈ duration 秒。默认 2s。
   */
  duration?: number
  className?: string
}

/**
 * 骨架屏 → 实际内容的交叉淡入/淡出。
 *
 * 实现选择：两个 motion.div 始终挂在 DOM 里，通过 CSS Grid 叠在同一格
 * （`col-start-1 row-start-1`），用 opacity 切换可见性，不走 AnimatePresence
 * 的 exit 状态机。这样 loading 一变，motion 内部靠 prop diff 直接驱动
 * opacity 过渡，行为可预测，**不会**出现"骨架屏立刻消失"的情况。
 *
 * - 严格尊重 `prefers-reduced-motion`：命中时直接硬切，零动画。
 * - 初次挂载不播放淡入（两个 motion.div 都用 `initial={false}`），符合
 *   "骨架屏就是初始占位"的预期；仅在 loading 由 true 变 false 时触发过渡。
 * - 容器高度 = max(skeleton, children)。调用方应保证两者高度接近，避免
 *   内容明显比骨架高/低时产生占位落差。
 */
export function SkeletonSwap({
  loading,
  skeleton,
  children,
  duration = 2,
  className,
}: SkeletonSwapProps) {
  const reduceMotion = useReducedMotion()
  const transition = { duration, ease: 'easeOut' as const }

  if (reduceMotion) {
    return <div className={className}>{loading ? skeleton : children}</div>
  }

  return (
    <div className={cn('grid', className)}>
      <motion.div
        className='col-start-1 row-start-1'
        initial={false}
        animate={{ opacity: loading ? 1 : 0 }}
        transition={transition}
        style={{ pointerEvents: loading ? 'auto' : 'none' }}
      >
        {skeleton}
      </motion.div>
      <motion.div
        className='col-start-1 row-start-1'
        initial={false}
        animate={{ opacity: loading ? 0 : 1 }}
        transition={transition}
        style={{ pointerEvents: loading ? 'none' : 'auto' }}
      >
        {children}
      </motion.div>
    </div>
  )
}

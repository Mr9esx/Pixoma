import { useEffect, useRef, useState } from 'react'
import { useReducedMotion } from 'motion/react'

type CountUpProps = {
  value: number
  /** 可选：对实时数值做格式化，默认四舍五入为整数。 */
  format?: (value: number) => string
  className?: string
}

/**
 * 数值变化时的 count-up 淡切换。
 * 首次挂载直接显示目标值（不做从 0 开始的出场动画），
 * 之后 value 变化时用 400ms easeOut 滚动到新值，让「状态变化」可感知。
 * 尊重 prefers-reduced-motion：命中时直接静态显示。
 */
export function CountUp({ value, format, className }: CountUpProps) {
  const reduceMotion = useReducedMotion()
  const [display, setDisplay] = useState(value)
  const lastValue = useRef(value)

  useEffect(() => {
    if (reduceMotion || lastValue.current === value) {
      setDisplay(value)
      lastValue.current = value
      return
    }

    const from = lastValue.current
    const to = value
    const startedAt = performance.now()
    const duration = 400
    let raf = 0

    const tick = (now: number) => {
      const progress = Math.min((now - startedAt) / duration, 1)
      const eased = 1 - Math.pow(1 - progress, 3)
      setDisplay(from + (to - from) * eased)
      if (progress < 1) {
        raf = requestAnimationFrame(tick)
      } else {
        lastValue.current = to
      }
    }

    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [value, reduceMotion])

  const text = format ? format(display) : String(Math.round(display))
  return <span className={className}>{text}</span>
}

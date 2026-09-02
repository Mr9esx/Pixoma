import { useLayoutEffect, useRef, type ReactNode } from 'react'
import { fitPhoneSize } from './fit-phone'

export function FitPhonePane({ children }: { children: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null)

  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    const apply = () => {
      const { width, height } = fitPhoneSize(el.clientWidth, el.clientHeight)
      if (width < 8 || height < 8) return
      el.style.setProperty('--phone-w', `${width}px`)
      el.style.setProperty('--phone-h', `${height}px`)
    }
    apply()
    const ro = new ResizeObserver(apply)
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  return (
    <div className='menu-puck-preview-pane flex h-full min-h-0 w-full overflow-hidden bg-muted p-3'>
      <div
        ref={ref}
        className='flex h-full min-h-0 w-full items-center justify-center overflow-hidden'
      >
        {children}
      </div>
    </div>
  )
}

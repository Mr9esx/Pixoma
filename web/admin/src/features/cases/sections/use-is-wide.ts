import { useEffect, useState } from 'react'

/** 宽屏（≥768px，Tailwind md）用表格，窄屏退回卡片布局。 */
export function useIsWide(): boolean {
  const [wide, setWide] = useState(() =>
    typeof window === 'undefined'
      ? false
      : window.matchMedia('(min-width: 768px)').matches
  )
  useEffect(() => {
    const m = window.matchMedia('(min-width: 768px)')
    const onChange = () => setWide(m.matches)
    onChange()
    m.addEventListener('change', onChange)
    return () => m.removeEventListener('change', onChange)
  }, [])
  return wide
}

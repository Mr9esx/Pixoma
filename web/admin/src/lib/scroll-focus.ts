import { useEffect } from 'react'

const FOCUS_KEY = 'pixoma-focus'

/** 跨页跳转前记录要定位的元素 id，目标页挂载后消费。 */
export function requestFocus(id: string) {
  try {
    sessionStorage.setItem(FOCUS_KEY, id)
  } catch {
    // storage 不可用时静默降级
  }
}

/** 滚动到目标元素并闪烁 3s；元素未就绪时最多重试约 2s。 */
export function scrollAndFlash(id: string, delay = 0) {
  const start = (attempt: number) => {
    const el = document.getElementById(id)
    if (!el) {
      if (attempt < 15) window.setTimeout(() => start(attempt + 1), 150)
      return
    }
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    el.classList.remove('flash-highlight')
    void el.offsetWidth
    el.classList.add('flash-highlight')
    window.setTimeout(() => el.classList.remove('flash-highlight'), 3300)
  }
  window.setTimeout(() => start(0), delay)
}

/** 页面挂载时消费上次跳转记录的目标并定位闪烁。 */
export function useScrollFocus() {
  useEffect(() => {
    let id: string | null = null
    try {
      id = sessionStorage.getItem(FOCUS_KEY)
      if (id) sessionStorage.removeItem(FOCUS_KEY)
    } catch {
      // storage 不可用时静默降级
    }
    if (id) scrollAndFlash(id)
  }, [])
}

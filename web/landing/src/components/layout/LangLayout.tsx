import { useEffect } from 'react'
import { Navigate, Outlet, useParams } from 'react-router-dom'
import i18n from '@/i18n'
import { isLangSegment, langSegmentToLocale } from '@/router/segments'

export function LangLayout() {
  const { lang } = useParams()
  const locale = langSegmentToLocale[lang ?? 'cn'] ?? 'zh-CN'
  const valid = isLangSegment(lang)

  useEffect(() => {
    if (!valid) return
    void i18n.changeLanguage(locale)
    document.documentElement.lang = locale
  }, [valid, locale])

  if (!valid) {
    return <Navigate to="/cn" replace />
  }

  return <Outlet />
}

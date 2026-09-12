import i18n from 'i18next'

export { i18n }
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import zh from './locales/zh.json'

export const LOCALE_STORAGE_KEY = 'admin-locale:v1'
export type AppLocale = 'zh' | 'en'

export function getStoredLocale(): AppLocale {
  const v = localStorage.getItem(LOCALE_STORAGE_KEY)
  return v === 'en' ? 'en' : 'zh'
}

export function setStoredLocale(locale: AppLocale): void {
  localStorage.setItem(LOCALE_STORAGE_KEY, locale)
}

export function localeToHtmlLang(locale: AppLocale): string {
  return locale === 'en' ? 'en' : 'zh-CN'
}

export function applyDocumentLang(locale: AppLocale): void {
  if (typeof document === 'undefined') return
  const lang = localeToHtmlLang(locale)
  document.documentElement.lang = lang
  const desc =
    locale === 'en'
      ? (en as { meta?: { description?: string } }).meta?.description
      : (zh as { meta?: { description?: string } }).meta?.description
  if (desc) {
    const meta = document.querySelector('meta[name="description"]')
    if (meta) meta.setAttribute('content', desc)
    for (const attr of ['og:description', 'twitter:description'] as const) {
      const el = document.querySelector(`meta[property="${attr}"]`)
      if (el) el.setAttribute('content', desc)
    }
  }
}

export async function initI18n() {
  const lng = typeof window !== 'undefined' ? getStoredLocale() : 'zh'
  await i18n.use(initReactI18next).init({
    resources: {
      zh: { translation: zh },
      en: { translation: en },
    },
    lng,
    fallbackLng: 'zh',
    interpolation: { escapeValue: false },
  })
  applyDocumentLang(lng)
  i18n.on('languageChanged', (next) => {
    applyDocumentLang(next.startsWith('en') ? 'en' : 'zh')
  })
  return i18n
}

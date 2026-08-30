import i18n from 'i18next'
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
  return i18n
}

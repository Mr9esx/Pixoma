import { type SupportedLocale } from '@/i18n'

export const langSegmentToLocale: Record<string, SupportedLocale> = {
  cn: 'zh-CN',
  en: 'en',
}

export const supportedLangSegments = ['cn', 'en'] as const
export type LangSegment = (typeof supportedLangSegments)[number]

export function isLangSegment(v: unknown): v is LangSegment {
  return typeof v === 'string' && supportedLangSegments.includes(v as LangSegment)
}

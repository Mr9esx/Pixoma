import { describe, expect, it } from 'vitest'
import { initI18n, resolveLocale } from '@/i18n'
import zhCN from '@/i18n/locales/zh-CN'
import en from '@/i18n/locales/en'

describe('i18n', () => {
  it('zh-CN 与 en 词条 key 完全对齐', () => {
    expect(Object.keys(zhCN).sort()).toEqual(Object.keys(en).sort())
  })

  it('默认语言为 zh-CN，可切换到 en', async () => {
    const i = await initI18n()
    expect(i.language).toBe('zh-CN')
    await i.changeLanguage('en')
    expect(i.language).toBe('en')
  })

  it('非法语言段回退 zh-CN', () => {
    expect(resolveLocale('invalid')).toBe('zh-CN')
    expect(resolveLocale(undefined)).toBe('zh-CN')
    expect(resolveLocale('cn')).toBe('zh-CN')
    expect(resolveLocale('en')).toBe('en')
  })
})

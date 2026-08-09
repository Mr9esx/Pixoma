import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, it, expect, beforeEach } from 'vitest'
import { getStoredLocale, setStoredLocale, LOCALE_STORAGE_KEY } from './index'
import zh from './locales/zh.json'
import en from './locales/en.json'

const COMMAND_MENU_I18N_KEYS = [
  'common.commandPlaceholder',
  'common.commandEmpty',
  'common.commandNavigation',
  'common.commandTheme',
  'theme.light',
  'theme.dark',
  'theme.system',
] as const

function lookup(locale: Record<string, unknown>, dottedKey: string): unknown {
  return dottedKey.split('.').reduce<unknown>((node, part) => {
    if (node == null || typeof node !== 'object') return undefined
    return (node as Record<string, unknown>)[part]
  }, locale)
}

function installLocalStorageMock() {
  const store = new Map<string, string>()
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => store.get(key) ?? null,
      setItem: (key: string, value: string) => {
        store.set(key, String(value))
      },
      removeItem: (key: string) => {
        store.delete(key)
      },
      clear: () => {
        store.clear()
      },
    },
  })
}

installLocalStorageMock()

beforeEach(() => {
  localStorage.clear()
})

describe('locale persistence', () => {
  it('defaults to zh', () => {
    expect(getStoredLocale()).toBe('zh')
  })

  it('persists en', () => {
    setStoredLocale('en')
    expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('en')
    expect(getStoredLocale()).toBe('en')
  })
})

describe('command menu shell i18n', () => {
  it('defines command/theme keys in zh and en locales', () => {
    for (const key of COMMAND_MENU_I18N_KEYS) {
      expect(lookup(zh, key), `zh missing ${key}`).toEqual(expect.any(String))
      expect(lookup(en, key), `en missing ${key}`).toEqual(expect.any(String))
    }
  })

  it('command-menu wraps shell strings with t()', () => {
    const here = dirname(fileURLToPath(import.meta.url))
    const source = readFileSync(
      join(here, '../../components/command-menu.tsx'),
      'utf8'
    )
    for (const key of COMMAND_MENU_I18N_KEYS) {
      expect(source).toContain(`t('${key}')`)
    }
  })
})

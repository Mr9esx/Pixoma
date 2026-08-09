import { describe, it, expect, beforeEach } from 'vitest'
import { getStoredLocale, setStoredLocale, LOCALE_STORAGE_KEY } from './index'

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

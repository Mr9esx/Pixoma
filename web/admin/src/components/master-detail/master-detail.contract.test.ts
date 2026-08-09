import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import en from '../../lib/i18n/locales/en.json'
import zh from '../../lib/i18n/locales/zh.json'

const here = dirname(fileURLToPath(import.meta.url))
const srcRoot = join(here, '../..')

const SHELL = join(here, 'master-detail-shell.tsx')
const EMPTY = join(srcRoot, 'components/feedback/empty-state.tsx')
const ERROR = join(srcRoot, 'components/feedback/error-banner.tsx')
const LOADING = join(srcRoot, 'components/feedback/loading-skeleton.tsx')
const INSTANCES_ROUTE = join(srcRoot, 'routes/_app/instances/route.tsx')

const FEEDBACK_I18N_KEYS = [
  'common.selectItem',
  'common.backToList',
  'common.empty',
  'common.loading',
  'common.retry',
  'common.errorGeneric',
] as const

function lookup(locale: Record<string, unknown>, dottedKey: string): unknown {
  return dottedKey.split('.').reduce<unknown>((node, part) => {
    if (node == null || typeof node !== 'object') return undefined
    return (node as Record<string, unknown>)[part]
  }, locale)
}

describe('Master–Detail shell + feedback primitives', () => {
  it('defines feedback/layout locale keys in zh and en', () => {
    for (const key of FEEDBACK_I18N_KEYS) {
      expect(lookup(zh, key), `zh missing ${key}`).toEqual(expect.any(String))
      expect(lookup(en, key), `en missing ${key}`).toEqual(expect.any(String))
    }
  })

  it('exports MasterDetailShell with layout B responsive split', () => {
    expect(existsSync(SHELL), 'master-detail-shell.tsx missing').toBe(true)
    const source = readFileSync(SHELL, 'utf8')
    expect(source).toContain('export function MasterDetailShell')
    expect(source).toContain('md:grid-cols-[minmax(280px,360px)_1fr]')
    expect(source).toContain('min-h-0')
    expect(source).toContain('flex-1')
    expect(source).not.toContain('100vh-5rem')
    expect(source).toContain("t('common.selectItem')")
    expect(source).toMatch(/t\('common\.backToList'/)
    expect(source).toContain('hasSelection')
    expect(source).toContain('onBackToList')
  })

  it('exports EmptyState / ErrorBanner / LoadingSkeleton with i18n', () => {
    expect(existsSync(EMPTY), 'empty-state.tsx missing').toBe(true)
    expect(existsSync(ERROR), 'error-banner.tsx missing').toBe(true)
    expect(existsSync(LOADING), 'loading-skeleton.tsx missing').toBe(true)

    const empty = readFileSync(EMPTY, 'utf8')
    const error = readFileSync(ERROR, 'utf8')
    const loading = readFileSync(LOADING, 'utf8')

    expect(empty).toContain('export function EmptyState')
    expect(empty).toContain("t('common.empty')")

    expect(error).toContain('export function ErrorBanner')
    expect(error).toContain("t('common.retry')")

    expect(loading).toContain('export function LoadingSkeleton')
    expect(loading).toMatch(/t\('common\.loading'|animate-pulse/)
  })

  it('instances layout route mounts MasterDetailShell', () => {
    const source = readFileSync(INSTANCES_ROUTE, 'utf8')
    expect(source).toContain('MasterDetailShell')
  })
})

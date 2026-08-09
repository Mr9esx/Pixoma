import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const API = join(here, '../../lib/api/tg-menu.ts')
const EDITOR = join(here, 'menu-editor.tsx')
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('tg menu tree editor', () => {
  it('api types expose tree MenuNode with intro_text', () => {
    const source = read(API)
    expect(source).toContain("export type MenuKind =")
    expect(source).toContain("kind: MenuKind")
    expect(source).toContain('case_ids?: string[]')
    expect(source).toContain('children?: MenuNode[]')
    expect(source).toContain('intro_text?: string')
    expect(source).toContain('export type TgMenuTree')
    expect(source).not.toContain('MenuAction')
    expect(source).not.toContain('action:')
  })

  it('editor uses expandable tree and folder intro fields', () => {
    const source = read(EDITOR)
    expect(source).toContain('visibleTreeRows')
    expect(source).toContain('expandedIds')
    expect(source).toContain('ancestorIds')
    expect(source).toContain("paddingLeft: `${8 + depth * 16}px`")
    expect(source).toContain('parentIsFolder')
    expect(source).toContain('fieldIntro')
    expect(source).toContain('fieldChildren')
    expect(source).toContain('intro_text')
    expect(source).toContain('addChildToTree')
    expect(source).toContain('onSelectChild')
  })

  it('i18n has plain-language folder intro keys', () => {
    const zh = JSON.parse(read(ZH)) as { tgMenu: Record<string, string> }
    const en = JSON.parse(read(EN)) as { tgMenu: Record<string, string> }
    for (const locale of [zh.tgMenu, en.tgMenu]) {
      expect(locale.listTitle).toBeTruthy()
      expect(locale.fieldIntro).toBeTruthy()
      expect(locale.fieldIntroHint).toBeTruthy()
      expect(locale.fieldCaseIds).toBeTruthy()
      expect(locale.fieldChildren).toBeTruthy()
      expect(locale.addChild).toBeTruthy()
      expect(locale.kindFolder).toBeTruthy()
    }
    expect(zh.tgMenu.listTitle).toBe('目录')
    expect(zh.tgMenu.fieldIntro).toBe('本层说明')
    expect(zh.tgMenu.fieldCaseIds).toBe('本层模板')
    expect(zh.tgMenu.fieldChildren).toBe('下面的分类')
    expect(zh.tgMenu.addChild).toBe('加一个分类')
  })
})

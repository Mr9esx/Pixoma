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
  it('api types expose tree MenuNode and MenuKind', () => {
    const source = read(API)
    expect(source).toContain("export type MenuKind =")
    expect(source).toContain("kind: MenuKind")
    expect(source).toContain('case_ids?: string[]')
    expect(source).toContain('children?: MenuNode[]')
    expect(source).toContain('export type TgMenuTree')
    expect(source).not.toContain('MenuAction')
    expect(source).not.toContain('action:')
  })

  it('editor renders indented tree and folder child actions', () => {
    const source = read(EDITOR)
    expect(source).toContain('flattenTree')
    expect(source).toContain("paddingLeft: `${16 + depth * 16}px`")
    expect(source).toContain('parentIsFolder')
    expect(source).toContain("kind === 'folder' ? 'folder'")
    expect(source).toContain('addChildToTree')
    expect(source).toContain('fieldCaseIds')
    expect(source).toContain('addChild')
  })

  it('i18n has folder / case mount / child keys', () => {
    const zh = JSON.parse(read(ZH)) as { tgMenu: Record<string, string> }
    const en = JSON.parse(read(EN)) as { tgMenu: Record<string, string> }
    expect(zh.tgMenu.kindFolder).toBeTruthy()
    expect(zh.tgMenu.fieldCaseIds).toBeTruthy()
    expect(zh.tgMenu.addChild).toBeTruthy()
    expect(en.tgMenu.kindFolder).toBeTruthy()
    expect(en.tgMenu.fieldCaseIds).toBeTruthy()
    expect(en.tgMenu.addChild).toBeTruthy()
  })
})

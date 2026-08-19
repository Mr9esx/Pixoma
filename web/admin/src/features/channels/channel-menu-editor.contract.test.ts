import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const API = join(here, '../../lib/api/channel-menu.ts')
const EDITOR = join(here, 'channel-menu-editor.tsx')
const PARAMS_FORM = join(here, 'params-form.tsx')
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')

function read(path: string) {
  return readFileSync(path, 'utf8')
}

describe('channel menu tree editor', () => {
  it('api types expose tree MenuNode with intro_text', () => {
    const source = read(API)
    expect(source).toContain('capability_id?: string')
    expect(source).toContain('params?: Record<string, unknown>')
    expect(source).toContain('render_override?: Record<string, unknown>')
    expect(source).toContain('children?: MenuNode[]')
    expect(source).toContain('intro_text?: string')
    expect(source).toContain('export type ChannelMenuTree')
    expect(source).toContain('order')
    expect(source).not.toContain('MenuAction')
    expect(source).not.toContain('action:')
  })

  it('editor uses expandable tree and folder intro fields', () => {
    const source = read(EDITOR)
    expect(source).toContain('visibleTreeRows')
    expect(source).toContain('expandedIds')
    expect(source).toContain('ancestorIds')
    expect(source).toContain('paddingLeft: `${8 + depth * 16}px`')
    expect(source).toContain('parentIsFolder')
    expect(source).toContain('fieldIntro')
    expect(source).toContain('fieldChildren')
    expect(source).toContain('intro_text')
    expect(source).toContain('addChildToTree')
    expect(source).toContain('onSelectChild')
  })

  it('i18n has plain-language folder intro keys', () => {
    const zh = JSON.parse(read(ZH)) as { channelMenu: Record<string, string> }
    const en = JSON.parse(read(EN)) as { channelMenu: Record<string, string> }
    for (const locale of [zh.channelMenu, en.channelMenu]) {
      expect(locale.listTitle).toBeTruthy()
      expect(locale.fieldIntro).toBeTruthy()
      expect(locale.fieldIntroHint).toBeTruthy()
      expect(locale.fieldCaseIds).toBeTruthy()
      expect(locale.fieldChildren).toBeTruthy()
      expect(locale.addChild).toBeTruthy()
      expect(locale.kindFolder).toBeTruthy()
    }
    expect(zh.channelMenu.listTitle).toBe('目录')
    expect(zh.channelMenu.fieldIntro).toBe('本层说明')
    expect(zh.channelMenu.fieldCaseIds).toBe('本层模板')
    expect(zh.channelMenu.fieldChildren).toBe('下面的分类')
    expect(zh.channelMenu.addChild).toBe('加一个分类')
  })
})

const NEW_KEYS = [
  'userView',
  'currentEditing',
  'buttonLabel',
  'buttonAction',
  'actionShowChildren',
  'actionOpenWorkflow',
  'actionReplyText',
  'actionReplyMedia',
  'paramSelectWorkflows',
  'paramReplyText',
  'paramReplyImages',
  'mainKeyboard',
  'columnsPerRow',
  'maxRootHint',
  'oneLevelHint',
  'unsavedCount',
  'invalidMainKeyboard',
  'invalidMissingAction',
  'invalidOpenCaseEmpty',
  'invalidNameEmpty',
  'simNote',
] as const

describe('wysiwyg menu editor i18n', () => {
  it('zh and en define every editor key', () => {
    const zh = JSON.parse(read(ZH)) as { channelMenu: Record<string, string> }
    const en = JSON.parse(read(EN)) as { channelMenu: Record<string, string> }
    for (const key of NEW_KEYS) {
      expect(zh.channelMenu[key], `zh missing channelMenu.${key}`).toBeTruthy()
      expect(en.channelMenu[key], `en missing channelMenu.${key}`).toBeTruthy()
    }
  })
})

describe('params form', () => {
  it('is schema driven with workflow picker widget', () => {
    const source = read(PARAMS_FORM)
    expect(source).toContain("data-testid='capability-params-form'")
    expect(source).toContain('workflow_picker')
    expect(source).toContain('x-admin')
    expect(source).toContain('caseOptions')
  })
})

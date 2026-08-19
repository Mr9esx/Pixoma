import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')

const NEW_KEYS = [
  'mainKeyboard',
  'columnsPerRow',
  'addMenuItem',
  'menuItemLabel',
  'menuItemAction',
  'cardName',
  'cardMedia',
  'cardText',
  'cardButtons',
  'addCardButton',
  'buttonLabel',
  'buttonAction',
  'actionOpenCard',
  'actionOpenWorkflow',
  'actionSendText',
  'actionSendMedia',
  'actionOpenUrl',
  'actionCopyText',
  'actionPlaceholder',
  'targetCard',
  'newCard',
  'pickExistingCard',
  'workflowList',
  'backTo',
  'cardList',
  'saveValidation',
  'errLabel',
  'errCard',
  'errWorkflow',
  'errUrl',
  'errMedia',
] as const

describe('menu editor i18n', () => {
  it('zh and en define every key', () => {
    const zh = JSON.parse(readFileSync(ZH, 'utf8')) as {
      menu: Record<string, string>
    }
    const en = JSON.parse(readFileSync(EN, 'utf8')) as {
      menu: Record<string, string>
    }
    for (const key of NEW_KEYS) {
      expect(zh.menu[key], `zh missing menu.${key}`).toBeTruthy()
      expect(en.menu[key], `en missing menu.${key}`).toBeTruthy()
    }
  })
})

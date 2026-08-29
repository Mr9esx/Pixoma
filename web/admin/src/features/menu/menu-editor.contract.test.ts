import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')
const ACTION_FORM = join(here, 'action-form.tsx')
const MENU_EDITOR = join(here, 'menu-card-editor.tsx')
const MENU_EDITOR_MODAL = join(here, 'menu-editor-modal.tsx')

const V2_KEYS = [
  'tabOutline',
  'tabLibrary',
  'structure',
  'preview',
  'editorEmptyHint',
  'workflowSingleHint',
  'addMenuItem',
  'newCard',
  'newCardTitle',
  'newCardName',
  'newCardText',
  'newCardConfirm',
  'deleteCard',
  'deleteKey',
  'deleteButton',
  'keyName',
  'addMedia',
  'previewNotSaved',
  'actionGroupWorkflow',
  'actionGroupTg',
  'libraryPinUsed',
  'libraryPinFree',
  'libraryBtnCount',
  'libraryNoText',
  'libraryReferrer',
  'outlineEmptyHint',
  'cardName',
  'cardText',
  'cardMedia',
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
  'menuItemLabel',
  'menuItemAction',
  'targetCard',
  'pickExistingCard',
  'workflowList',
  'saveValidation',
  'errLabel',
  'errCard',
  'errWorkflow',
  'errUrl',
  'errMedia',
  'unsavedCount',
  'untitled',
  'editMenu',
  'editMenuTitle',
  'editMenuHint',
  'menuItemCount',
  'detailTitle',
  'detailPickHint',
  'kind_item',
  'kind_card',
  'kind_button',
  'mapPath',
  'mapEmpty',
  'mapOrphans',
  'mapCountKeys',
  'mapCountCards',
  'mapCountWorkflows',
  'mapOpenCard',
  'mapStartWorkflow',
  'mapSendText',
  'mapSendMedia',
  'mapOpenUrl',
  'mapCopyText',
  'mapCardMissing',
  'mapWorkflowMissing',
  'workflowCardView',
  'mapPayloadWorkflow',
  'mapPayloadText',
  'mapPayloadCopy',
  'mapPayloadMedia',
  'mapPayloadUrl',
  'mapThumbImage',
  'mapThumbVideo',
  'mapThumbAnimation',
  'mapCrumbOrphan',
  'mapLoadFailed',
  'columnCount',
  'addKey',
  'actionRow',
  'unsaved',
  'discardEdits',
  'discard',
] as const

const REMOVED_V1_KEYS = [
  'openModeList',
  'openModeDirect',
  'openModeListAdvanced',
  'actionPlaceholder',
  'mainKeyboard',
  'columnsPerRow',
  'cardList',
  'newCard',
] as const

describe('menu editor i18n (v2)', () => {
  it('zh and en define every v2 key', () => {
    const zh = JSON.parse(readFileSync(ZH, 'utf8')) as {
      menu: Record<string, string>
    }
    const en = JSON.parse(readFileSync(EN, 'utf8')) as {
      menu: Record<string, string>
    }
    for (const key of V2_KEYS) {
      expect(zh.menu[key], `zh missing menu.${key}`).toBeTruthy()
      expect(en.menu[key], `en missing menu.${key}`).toBeTruthy()
    }
  })

  it('v1 placeholder / openMode keys are removed', () => {
    const zh = JSON.parse(readFileSync(ZH, 'utf8')) as {
      menu: Record<string, string>
    }
    const en = JSON.parse(readFileSync(EN, 'utf8')) as {
      menu: Record<string, string>
    }
    for (const key of REMOVED_V1_KEYS) {
      // 'newCard' 在 v2 里是新建卡片按钮文案，留着，单独从 removed 列表里剔除
      if (key === 'newCard' || key === 'mainKeyboard' || key === 'cardList')
        continue
      expect(zh.menu[key], `zh should not have menu.${key}`).toBeUndefined()
      expect(en.menu[key], `en should not have menu.${key}`).toBeUndefined()
    }
  })

  it('editMenu is 改菜单', () => {
    const zh = JSON.parse(readFileSync(ZH, 'utf8')) as {
      menu: Record<string, string>
    }
    expect(zh.menu.editMenu).toBe('改菜单')
  })

  it('writable map chrome copy', () => {
    const zh = JSON.parse(readFileSync(ZH, 'utf8')) as {
      menu: Record<string, string>
    }
    expect(zh.menu.columnCount).toBe('每行')
    expect(zh.menu.addKey).toBe('+ 加键')
    expect(zh.menu.actionRow).toBe('动作')
    expect(zh.menu.keyName).toBe('键名')
    expect(zh.menu.deleteKey).toBe('删这个键')
    expect(zh.menu.deleteButton).toBe('删这个按钮')
    expect(zh.menu.deleteCard).toBe('删这张卡片')
    expect(zh.menu.unsaved).toBe('未保存')
    expect(zh.menu.discardEdits).toBe('丢弃这次修改？')
    expect(zh.menu.discard).toBe('丢弃')
    expect(zh.menu.editMenu).toBe('改菜单')
  })
})

describe('action form (v2)', () => {
  it('uses single Select + optgroup + 6 action types', () => {
    const source = readFileSync(ACTION_FORM, 'utf8')
    expect(source).toContain("data-testid='action-form'")
    expect(source).toContain("data-testid='action-type-select'")
    expect(source).toContain('SelectGroup')
    expect(source).toContain('SelectLabel')
    expect(source).toContain("t('menu.actionGroupWorkflow')")
    expect(source).toContain("t('menu.actionGroupTg')")
    for (const key of [
      'open_card',
      'open_workflow',
      'send_text',
      'send_media',
      'open_url',
      'copy_text',
    ]) {
      expect(source).toContain(`action.type === '${key}'`)
    }
    expect(source).toContain('workflow_id')
    expect(source).not.toContain('workflow_ids')
    expect(source).not.toContain('WorkflowMenuMode')
    expect(source).not.toContain('打开方式')
    expect(source).not.toContain('actionPlaceholder')
  })

  it('renders workflow info card when a workflow is bound', () => {
    const source = readFileSync(ACTION_FORM, 'utf8')
    expect(source).toContain(
      "import { WorkflowInfoCard } from './workflow-info-card'"
    )
    expect(source).toContain('WorkflowInfoCard')
    expect(source).toMatch(
      /action\.type === 'open_workflow' && action\.workflow_id/
    )
  })

  it('workflow info card renders the preview via the auth media loader', () => {
    const card = readFileSync(join(here, 'workflow-info-card.tsx'), 'utf8')
    expect(card).toContain('useMediaObjectUrl')
    expect(card).toContain('resolveMediaKey')
    expect(card).not.toContain('src={workflow.preview}')
  })

  it('workflow info card opens the workflow page in a new tab', () => {
    const card = readFileSync(join(here, 'workflow-info-card.tsx'), 'utf8')
    expect(card).toContain("target='_blank'")
    expect(card).toContain('href={`/cases/${workflow.id}`}')
    expect(card).toContain("t('menu.workflowCardView')")
    expect(card).not.toContain('disabled')
  })
})

describe('menu view mode (主视图只读能力地图)', () => {
  it('hosts MenuCapabilityMap and MenuEditorModal only', () => {
    const source = readFileSync(MENU_EDITOR, 'utf8')
    expect(source).toContain("data-testid='menu-card-editor'")
    expect(source).toContain('MenuCapabilityMap')
    expect(source).toContain('MenuEditorModal')
    expect(source).toContain('mapLoadFailed')
    expect(source).not.toContain('PhoneSimulation')
    expect(source).not.toContain("data-testid='left-pane'")
    expect(source).not.toContain("data-testid='preview-pane'")
    expect(source).not.toContain("data-testid='tab-outline'")
    expect(source).not.toContain("data-testid='tab-library'")
    expect(source).not.toContain('OutlinePane')
    expect(source).not.toContain('CardLibraryPane')
    expect(source).not.toContain("data-testid='add-menu-item'")
    expect(source).not.toContain("data-testid='save-menu'")
  })
})

describe('menu edit mode (可写能力地图弹层)', () => {
  it('is a writable map: keyboard + path + save, no outline/library/phone', () => {
    const source = readFileSync(MENU_EDITOR_MODAL, 'utf8')
    const writable = readFileSync(join(here, 'menu-writable-map.tsx'), 'utf8')
    const layout = readFileSync(join(here, 'menu-map-layout.tsx'), 'utf8')
    const all = source + writable + layout
    expect(source).toContain("data-testid='menu-editor-modal'")
    expect(source).toContain("data-testid='save-menu'")
    expect(source).toContain('h-svh')
    expect(source).not.toContain("data-testid='menu-columns'")
    expect(all).toContain("data-testid='menu-columns'")
    expect(all).toContain("data-testid='add-key'")
    expect(all).toContain("data-testid='map-keyboard'")
    expect(all).toContain("data-testid='map-path'")
    expect(source).toContain('validateMenuConfig')
    expect(source).toContain('putMenu')
    expect(source).toContain('onSaved')
    expect(source).toContain('isMenuDraftDirty')
    expect(source).toContain('discardEdits')
    expect(source).toContain('AlertDialog')
    expect(source).toContain("t('menu.discard')")
    expect(source).not.toContain("data-testid='tab-outline'")
    expect(source).not.toContain("data-testid='tab-library'")
    expect(source).not.toContain("data-testid='phone-simulation'")
    expect(source).not.toContain("data-testid='new-card'")
    expect(source).not.toContain("data-testid='editor-pane'")
    expect(source).not.toContain('OutlinePane')
    expect(source).not.toContain('CardLibraryPane')
    expect(source).not.toContain('PhoneSimulation')
    expect(source).not.toContain('NewCardDialog')
    expect(source).not.toContain('ItemEditor')
    expect(source).not.toContain('CardEditor')
    expect(source).not.toContain('ButtonEditor')
  })

  it('dead editor panes are gone', () => {
    const files = [
      'outline-pane.tsx',
      'card-library-pane.tsx',
      'phone-simulation.tsx',
      'new-card-dialog.tsx',
      'item-editor.tsx',
      'card-editor.tsx',
      'button-editor.tsx',
    ]
    for (const f of files) {
      expect(existsSync(join(here, f))).toBe(false)
    }
  })
})

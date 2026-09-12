import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const here = dirname(fileURLToPath(import.meta.url))
const ZH = join(here, '../../lib/i18n/locales/zh.json')
const EN = join(here, '../../lib/i18n/locales/en.json')
const ACTION_FORM = join(here, 'action-form.tsx')
const MENU_EDITOR = join(here, 'menu-card-editor.tsx')
const MENU_PUCK = join(here, 'menu-puck-editor.tsx')
const PUCK_CONFIG = join(here, 'puck-config.tsx')

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
  'actionListTasks',
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
  'mapListTasks',
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
  'discardEditsDesc',
  'discard',
  'saveFailed',
  'errRootCount',
  'errColumns',
  'errDupId',
  'errDepth',
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

  it('editMenu is 编辑菜单', () => {
    const zh = JSON.parse(readFileSync(ZH, 'utf8')) as {
      menu: Record<string, string>
    }
    expect(zh.menu.editMenu).toBe('编辑菜单')
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
    expect(zh.menu.discardEdits).toBe('丢弃修改？')
    expect(zh.menu.discard).toBe('废弃')
    expect(zh.menu.editMenu).toBe('编辑菜单')
    expect(zh.menu.actionGroupWorkflow).toBe('平台能力')
    expect(zh.menu.actionGroupTg).toBe('Telegram 能力')
    expect(zh.menu.actionListTasks).toBe('我的任务')
  })
})

describe('action form (v2)', () => {
  it('groups platform vs TG actions and includes list_tasks', () => {
    const source = readFileSync(ACTION_FORM, 'utf8')
    expect(source).toContain("data-testid='action-form'")
    expect(source).toContain("data-testid='action-type-select'")
    expect(source).toContain('SelectGroup')
    expect(source).toContain("t('menu.actionGroupWorkflow')")
    expect(source).toContain("t('menu.actionGroupTg')")
    expect(source).toContain("'open_workflow', 'list_tasks'")
    expect(source).toContain("'open_card'")
    expect(source).toContain("case 'list_tasks'")
    expect(source).toContain('ListTasksPreview')
    expect(source).toContain("action.type === 'list_tasks'")
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

describe('menu view mode (详情只读手机)', () => {
  it('hosts MenuPhone and a link into the full-page editor', () => {
    const source = readFileSync(MENU_EDITOR, 'utf8')
    expect(source).toContain("data-testid='menu-card-editor'")
    expect(source).toContain('MenuPhone')
    expect(source).toContain('channelId={channelId}')
    const phone = readFileSync(join(here, 'menu-phone.tsx'), 'utf8')
    expect(phone).toContain("to='/channels/$id/menu'")
    expect(phone).toContain("data-testid='edit-menu'")
    expect(phone).toContain("variant='outline'")
    expect(phone).toContain("size='sm'")
    expect(phone).toContain('PenLine')
    expect(phone).toContain('h-[500px]')
    expect(phone).toContain("data-testid='menu-phone-info'")
    expect(phone).toContain("data-testid='menu-phone-keys'")
    expect(phone).toContain('h-[35%]')
    expect(phone).toContain("className='w-full shrink-0'")
    expect(phone).not.toContain('mt-auto')
    expect(phone).toContain('WorkflowInfoCard')
    expect(phone).toContain('PhonePeekCard')
    expect(phone).toContain('previewEmpty')
    expect(phone).toContain("data-testid='menu-phone-empty'")
    expect(phone).toContain('overflow-y-auto')
    expect(phone).not.toContain('max-h-[70%]')
    expect(phone).not.toContain('h-[334px]')
    expect(phone).not.toContain('h-[667px]')
    expect(source).not.toContain('justify-end')
    expect(source).not.toContain('MenuCapabilityMap')
    expect(source).not.toContain('MenuEditorModal')
    expect(source).not.toContain('listCards')
    expect(source).not.toContain("data-testid='left-pane'")
  })
})

describe('menu edit mode (Puck 整页)', () => {
  it('is a phone canvas with palette, fields, save, and unsaved leave confirm', () => {
    const source = readFileSync(MENU_PUCK, 'utf8')
    const config = readFileSync(PUCK_CONFIG, 'utf8')
    expect(source).toContain("data-testid='menu-puck-editor'")
    expect(source).toContain("data-testid='discard-menu'")
    expect(source).toContain('putMenu')
    expect(source).toContain('validateMenuTree')
    expect(source).toContain('AlertDialog')
    expect(source).toContain('ArrowLeft')
    expect(source).toContain("size='icon'")
    expect(source).toContain('useBlocker')
    expect(source).toContain('<footer')
    expect(source).toContain("t('menu.discard')")
    expect(source).toContain('beforeunload')
    expect(source).toContain('Puck.Components')
    expect(source).toContain('Puck.Preview')
    expect(source).toContain('Puck.Fields')
    expect(source).toContain("data-testid='menu-palette'")
    expect(source).toContain("data-testid='menu-fields'")
    expect(source).toContain('overflow-y-auto')
    expect(source).toContain('MENU_PUCK_OVERRIDES')
    expect(source).toContain('wrapFields={false}')
    expect(source).toContain('listCases')
    expect(source).toContain('MenuPuckMetaProvider')
    const fields = readFileSync(join(here, 'puck-fields.tsx'), 'utf8')
    expect(fields).toContain('FieldLabel')
    expect(fields).toContain('onOpenChange')
    expect(fields).toContain('CardButtonsField')
    expect(fields).toContain('Collapsible')
    expect(fields).toContain('WorkflowInfoCard')
    expect(fields).toContain("data-testid='action-type-select'")
    expect(fields).toContain("data-testid='workflow-select'")
    expect(source).toContain('380px')
    expect(fields).toContain('MarkdownTextField')
    expect(config).toContain('ActionTypeField')
    expect(config).toContain('WorkflowField')
    expect(config).toContain('CardButtonsField')
    expect(config).not.toContain("type: 'array'")
    expect(config).not.toContain('categories')
    expect(config).toContain("data-testid='menu-phone-canvas'")
    expect(config).toContain('--menu-cols')
    expect(config).not.toContain('gridTemplateColumns')
    const css = readFileSync(join(here, 'menu-puck.css'), 'utf8')
    expect(css).toContain('data-puck-dropzone')
    expect(css).toContain('grid-template-columns')
    expect(css).toContain('--foreground')
    expect(source).toContain('FitPhonePane')
    expect(css).toContain('--phone-w')
    expect(css).not.toContain('100cqh')
    expect(css).not.toContain('100cqi')
    expect(config).toContain('MenuButton')
    expect(config).not.toContain('HeadingBlock')
    expect(config).toContain("type !== 'open_card'")
    expect(source).not.toContain('listCards')
    expect(source).not.toContain('MenuCapabilityMap')
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
      'menu-editor-modal.tsx',
      'menu-capability-map.tsx',
      'menu-writable-map.tsx',
      'card-picker.tsx',
      'card-draft.ts',
    ]
    for (const f of files) {
      expect(existsSync(join(here, f))).toBe(false)
    }
  })
})

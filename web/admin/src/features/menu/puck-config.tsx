import type { Config } from '@measured/puck'
import type { ReactNode } from 'react'
import {
  ActionTypeField,
  ButtonLabelField,
  CardButtonsField,
  CardTextField,
  ColumnsField,
  ListTasksField,
  MediaField,
  TextPayloadField,
  UrlField,
  WorkflowField,
} from './puck-fields'

const actionTypeField = {
  type: 'custom' as const,
  label: '点击后做什么',
  render: ActionTypeField,
}

const workflowField = {
  type: 'custom' as const,
  label: '选择工作流',
  render: WorkflowField,
}

const textField = {
  type: 'custom' as const,
  label: '文字',
  render: TextPayloadField,
}

const urlField = {
  type: 'custom' as const,
  label: '链接',
  render: UrlField,
}

const mediaField = {
  type: 'custom' as const,
  label: '媒体 URL',
  render: MediaField,
}

const cardTextField = {
  type: 'custom' as const,
  label: '卡片文字',
  render: CardTextField,
}

const listPreviewField = {
  type: 'custom' as const,
  label: '预览',
  render: ListTasksField,
}

const buttonFields = {
  label: { type: 'custom' as const, label: '按钮名字', render: ButtonLabelField },
  actionType: actionTypeField,
  workflow_id: workflowField,
  text: textField,
  url: urlField,
  mediaUrl: mediaField,
  cardText: cardTextField,
  listPreview: listPreviewField,
  cardButtons: {
    type: 'custom' as const,
    label: '卡片按钮',
    render: CardButtonsField,
  },
}

function fieldsForAction(type: string): typeof buttonFields {
  const fields: Record<string, unknown> = { ...buttonFields }
  if (type !== 'open_workflow') delete fields.workflow_id
  if (type !== 'list_tasks') delete fields.listPreview
  if (type !== 'send_text' && type !== 'copy_text') delete fields.text
  if (type !== 'open_url') delete fields.url
  if (type !== 'send_media') delete fields.mediaUrl
  if (type !== 'open_card') {
    delete fields.cardText
    delete fields.cardButtons
  }
  return fields as typeof buttonFields
}

export const menuPuckConfig = {
  root: {
    fields: {
      columns: {
        type: 'custom' as const,
        label: '每行',
        render: ColumnsField,
      },
    },
    defaultProps: { columns: 2 },
    render: ({ children, columns }: { children: ReactNode; columns?: number }) => (
      <div
        data-testid='menu-phone-canvas'
        className='mx-auto flex flex-col overflow-hidden rounded-xl border bg-card'
        style={{ ['--menu-cols' as string]: Number(columns) || 2 }}
      >
        <div className='h-10 shrink-0 border-b bg-muted' />
        <div className='mt-auto p-3'>{children}</div>
      </div>
    ),
  },
  components: {
    MenuButton: {
      label: '菜单按钮',
      defaultProps: {
        id: '',
        label: '按钮',
        actionType: 'send_text',
        workflow_id: '',
        text: '',
        url: '',
        mediaUrl: '',
        cardText: '',
        listPreview: '',
        cardButtons: [],
      },
      fields: buttonFields,
      resolveFields: (data: { props: { actionType?: string } }) =>
        fieldsForAction(String(data.props.actionType || 'send_text')),
      render: (props: { label?: string }) => (
        <button
          type='button'
          className='h-11 w-full rounded-md border bg-secondary px-2 text-sm'
        >
          {props.label || '按钮'}
        </button>
      ),
    },
  },
} as unknown as Config

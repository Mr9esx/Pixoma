import { type ComponentProps } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import {
  PromptInput,
  PromptInputBody,
  PromptInputTextarea,
} from '@/components/ai-elements/prompt-input'
import { StudioActionPanel, StudioChat } from './studio-chat'

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  listStudioLibraryAssets: vi.fn(() => Promise.resolve([])),
}))

describe('StudioChat', () => {
  it('uses AI Elements to render transcript Markdown and the chat input', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioChat
          assets={[]}
          messages={[]}
          modelConfigId='model-1'
          models={[
            {
              agent_enabled: true,
              base_url: 'https://example.com',
              capabilities: {
                image_output: false,
                streaming: true,
                tools: true,
                vision: false,
              },
              default: true,
              enabled: true,
              has_api_key: true,
              id: 'model-1',
              limits: {
                context_window_tokens: 128000,
                max_input_tokens: 32000,
                max_output_tokens: 8000,
              },
              model: 'pixoma-chat',
              name: 'Pixoma Chat',
              protocol: 'openai_responses',
              thinking: { enabled: true },
            },
          ]}
          onAssetChange={() => {}}
          onImportLibraryAsset={async () => {
            throw new Error('不应导入资产')
          }}
          onModelChange={() => {}}
          onPermissionChange={() => {}}
          onSkillChange={() => {}}
          permissionMode='request_approval'
          selectedAssets={[]}
          selectedSkillIds={[]}
          sessionId='session-1'
          skills={[]}
          transcript={{
            events: [],
            messages: [
              { content: '请给我一个大纲', id: 'user-1', role: 'user' },
              {
                content: '## 创作大纲\n\n- 设定\n- 冲突',
                id: 'assistant-1',
                role: 'assistant',
              },
              {
                content: '先整理主题和角色关系。',
                id: 'reasoning-1',
                role: 'reasoning',
              },
            ],
          }}
        />
      </QueryClientProvider>
    )

    await expect.element(screen.getByRole('log')).toBeVisible()
    await expect
      .element(screen.getByRole('heading', { name: '创作大纲' }))
      .toBeVisible()
    await expect.element(screen.getByText('思考过程')).toBeVisible()
    await expect
      .element(
        screen.getByPlaceholder('描述你想创作的内容，或让 Agent 调用工作流…')
      )
      .toBeVisible()
  })

  it('floats the composer over the conversation and reserves room for it', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioChat
          assets={[]}
          messages={[]}
          modelConfigId='model-1'
          models={[
            {
              agent_enabled: true,
              base_url: 'https://example.com',
              capabilities: {
                image_output: false,
                streaming: true,
                tools: true,
                vision: false,
              },
              default: true,
              enabled: true,
              has_api_key: true,
              id: 'model-1',
              limits: {
                context_window_tokens: 128000,
                max_input_tokens: 32000,
                max_output_tokens: 8000,
              },
              model: 'pixoma-chat',
              name: 'Pixoma Chat',
              protocol: 'openai_responses',
              thinking: { enabled: true },
            },
          ]}
          onAssetChange={() => {}}
          onImportLibraryAsset={async () => {
            throw new Error('不应导入资产')
          }}
          onModelChange={() => {}}
          onPermissionChange={() => {}}
          onSkillChange={() => {}}
          permissionMode='request_approval'
          selectedAssets={[]}
          selectedSkillIds={[]}
          sessionId='session-1'
          skills={[]}
          transcript={{
            events: [],
            messages: [
              { content: '请给我一个大纲', id: 'user-1', role: 'user' },
            ],
          }}
        />
      </QueryClientProvider>
    )

    const log = screen.getByRole('log').element()
    const composer = document.querySelector(
      '[data-slot="studio-composer"]'
    ) as HTMLElement
    expect(composer.parentElement).toBe(log.parentElement)
    expect(getComputedStyle(composer).position).toBe('absolute')
    expect(getComputedStyle(composer).bottom).toBe('0px')
    const content = log.querySelector('.pb-44') as HTMLElement
    expect(parseFloat(getComputedStyle(content).paddingBottom)).toBeGreaterThan(
      160
    )
    const group = document.querySelector(
      '[data-slot="input-group"]'
    ) as HTMLElement
    expect(getComputedStyle(group).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
    expect(getComputedStyle(group).borderTopLeftRadius).toBe('8px')
  })

  it('tucks the approval alert behind the composer input', async () => {
    const responded: Array<[string, boolean]> = []
    const screen = await render(
      <div className='mx-auto flex w-full max-w-3xl flex-col px-5'>
        <StudioActionPanel
          actions={[
            {
              id: 'interrupt-1',
              reason: 'tool_approval',
              message: '需要写入 Session 资产',
            },
          ]}
          onRespond={(id, approved) => responded.push([id, approved])}
        />
        <div className='pointer-events-auto relative z-10 pb-5'>
          <PromptInput inputGroupClassName='bg-background' onSubmit={() => {}}>
            <PromptInputBody>
              <PromptInputTextarea aria-label='输入' />
            </PromptInputBody>
          </PromptInput>
        </div>
      </div>
    )

    const panel = document.querySelector('[aria-label="操作区"]') as HTMLElement
    const alert = screen.getByRole('alert').element()
    const group = document.querySelector(
      '[data-slot="input-group"]'
    ) as HTMLElement
    const alertBox = alert.getBoundingClientRect()
    const groupBox = group.getBoundingClientRect()

    expect(getComputedStyle(panel).marginBottom).toBe('-16px')
    expect(groupBox.top - alertBox.bottom).toBe(-16)
    expect(alertBox.left - groupBox.left).toBe(8)
    expect(groupBox.right - alertBox.right).toBe(8)

    const title = alert.querySelector(
      '[data-slot="alert-title"]'
    ) as HTMLElement
    const description = alert.querySelector(
      '[data-slot="alert-description"]'
    ) as HTMLElement
    const approve = screen.getByRole('button', { name: /^批准$/ }).element()
    const titleBox = title.getBoundingClientRect()
    const descriptionBox = description.getBoundingClientRect()
    const approveBox = approve.getBoundingClientRect()
    expect(title.textContent).toBe('权限审批')
    expect(description.textContent).toBe('需要写入 Session 资产')
    // 右侧内边距 16px 加上 1px 边框
    expect(alertBox.right - approveBox.right).toBe(17)
    expect(descriptionBox.left - alertBox.left).toBe(17)
    expect(titleBox.bottom).toBeLessThan(descriptionBox.top)
    expect(descriptionBox.right).toBeLessThan(approveBox.left)
    expect(descriptionBox.top).toBeLessThan(approveBox.bottom)
    expect(approveBox.top).toBeLessThan(descriptionBox.bottom)
    for (const [side, value] of [
      ['paddingTop', '16px'],
      ['paddingRight', '16px'],
      ['paddingBottom', '32px'],
      ['paddingLeft', '16px'],
    ] as const) {
      expect(getComputedStyle(alert)[side]).toBe(value)
    }

    await expect
      .element(screen.getByRole('button', { name: /^批准$/ }))
      .toBeVisible()
    await screen.getByRole('button', { name: /^批准$/ }).click()
    await screen.getByRole('button', { name: /^拒绝$/ }).click()
    expect(responded).toEqual([
      ['interrupt-1', true],
      ['interrupt-1', false],
    ])
  })

  it('keeps the approval buttons beside a long pending action', async () => {
    const screen = await render(
      <div className='mx-auto flex w-full max-w-3xl flex-col px-5'>
        <StudioActionPanel
          actions={[
            {
              id: 'interrupt-long',
              reason: 'tool_approval',
              message:
                '把本轮生成的分镜脚本写入资产库，并同步更新故事板里的镜头顺序、角色出场安排与场景标记，覆盖原有的旧版本记录',
            },
          ]}
          onRespond={() => {}}
        />
      </div>
    )

    const alert = screen.getByRole('alert').element()
    const description = alert.querySelector(
      '[data-slot="alert-description"]'
    ) as HTMLElement
    const approve = screen.getByRole('button', { name: /^批准$/ }).element()
    const descriptionBox = description.getBoundingClientRect()
    const approveBox = approve.getBoundingClientRect()
    expect(descriptionBox.right).toBeLessThanOrEqual(approveBox.left)
    expect(descriptionBox.height).toBeGreaterThan(30)
    expect(alert.getBoundingClientRect().right - approveBox.right).toBe(17)
  })

  it('groups the model switcher with the send button and keeps Skills and assets as icon buttons', async () => {
    const onModelChange = vi.fn()
    const screen = await renderStudioChat({
      onModelChange,
      selectedAssets: [{ assetId: 'asset-1', assetVersionId: 'version-1' }],
      selectedSkillIds: ['skill-1'],
      skills: [
        {
          description: '把大纲写成镜头',
          enabled: true,
          id: 'skill-1',
          name: '分镜草稿',
        },
      ],
    })

    const group = document.querySelector(
      '[data-slot="input-group"]'
    ) as HTMLElement
    const inputRadius = getComputedStyle(group).borderTopLeftRadius
    expect(inputRadius).toBe('8px')

    const modelButton = screen
      .getByRole('button', { name: 'Pixoma Chat' })
      .element()
    const submit = screen.getByRole('button', { name: '发送消息' }).element()
    const footer = submit.closest(
      '[data-slot="input-group-addon"]'
    ) as HTMLElement
    const rightGroup = modelButton.parentElement as HTMLElement
    expect(modelButton.nextElementSibling).toBe(submit)
    expect(rightGroup).toBe(submit.parentElement)
    expect(footer.lastElementChild).toBe(rightGroup)

    const skillButton = screen
      .getByRole('button', { name: '选择 Skills，已选 1 项' })
      .element()
    const assetButton = screen
      .getByRole('button', { name: '选择资产，已选 1 项' })
      .element()
    const permissionButton = screen
      .getByRole('button', { name: 'Agent 操作权限：请求批准' })
      .element()
    const leftGroup = footer.firstElementChild as HTMLElement
    expect(leftGroup.children[0]).toBe(skillButton)
    expect(leftGroup.children[1]).toBe(assetButton)
    expect(leftGroup.children[2]).toBe(permissionButton)
    for (const button of [skillButton, assetButton]) {
      expect(button.textContent).toBe('')
      expect(getComputedStyle(button).borderTopLeftRadius).toBe(inputRadius)
    }
    for (const button of [modelButton, permissionButton]) {
      expect(getComputedStyle(button).fontWeight).toBe('400')
    }

    await screen.getByRole('button', { name: '选择 Skills，已选 1 项' }).hover()
    await expect
      .element(screen.getByRole('tooltip'))
      .toHaveTextContent('Skills · 1')

    await screen.getByRole('button', { name: 'Pixoma Chat' }).click()
    await expect.element(screen.getByRole('menu')).toBeVisible()
    expect(screen.getByRole('dialog').query()).toBeNull()
    await screen.getByRole('menuitemradio', { name: 'Pixoma Pro' }).click()
    expect(onModelChange).toHaveBeenCalledWith('model-2')
  })
})

const studioModels: ComponentProps<typeof StudioChat>['models'] = [
  studioModel('model-1', 'Pixoma Chat'),
  studioModel('model-2', 'Pixoma Pro'),
]

function studioModel(id: string, name: string) {
  return {
    agent_enabled: true,
    base_url: 'https://example.com',
    capabilities: {
      image_output: false,
      streaming: true,
      tools: true,
      vision: false,
    },
    default: id === 'model-1',
    enabled: true,
    has_api_key: true,
    id,
    limits: {
      context_window_tokens: 128000,
      max_input_tokens: 32000,
      max_output_tokens: 8000,
    },
    model: id,
    name,
    protocol: 'openai_responses' as const,
    thinking: { enabled: false },
  }
}

function renderStudioChat(
  overrides: Partial<ComponentProps<typeof StudioChat>> = {}
) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, refetchInterval: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <StudioChat
        assets={[]}
        messages={[]}
        modelConfigId='model-1'
        models={studioModels}
        onAssetChange={() => {}}
        onImportLibraryAsset={async () => {
          throw new Error('不应导入资产')
        }}
        onModelChange={() => {}}
        onPermissionChange={() => {}}
        onSkillChange={() => {}}
        permissionMode='request_approval'
        selectedAssets={[]}
        selectedSkillIds={[]}
        sessionId='session-1'
        skills={[]}
        transcript={{ events: [], messages: [] }}
        {...overrides}
      />
    </QueryClientProvider>
  )
}

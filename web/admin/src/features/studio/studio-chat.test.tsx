import { type ComponentProps } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { beforeAll, describe, expect, it, vi } from 'vitest'
import { userEvent } from 'vitest/browser'
import { render } from 'vitest-browser-react'
import { StudioChat } from './studio-chat'
import { initI18n } from '@/lib/i18n'

beforeAll(initI18n)

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  listStudioLibraryAssets: vi.fn(() =>
    Promise.resolve({ assets: [], total: 0 })
  ),
}))

describe('StudioChat', () => {
  it('shows a workflow input card and hides the message input', async () => {
    const screen = await renderStudioChat({
      latestRun: {
        created_at: '2026-09-26T10:00:00Z', id: 'run-1', session_id: 'session-1',
        status: 'waiting_clarification', trigger_message_id: 'user-1',
      },
      pendingClarifications: [{
        id: 'workflow-input-1', reason: 'workflow_input', message: '角色三视图',
        metadata: {
          messageId: 'run-1:clarification:workflow-input-1',
          workflow: {
            id: '12', name: '角色三视图', input_schema: {
              type: 'object', properties: { prompt: { type: 'string', title: '角色描述' } }, required: ['prompt'],
            },
            input_fields: [{ key: 'prompt', type: 'string', required: true }],
          },
        },
      }],
    })

    await expect.element(screen.getByTestId('studio-workflow-card')).toBeVisible()
    await expect
      .element(screen.getByRole('status'))
      .toHaveTextContent('等待你填写工作流参数')
    expect(document.querySelector<HTMLElement>("[data-slot='studio-composer']")?.className).toContain('hidden')
    await expect.element(screen.getByRole('textbox', { name: '角色描述 *' })).toBeDisabled()
  })

  it('sends a typed user message with the selected run configuration and renders the streamed reply', async () => {
    const requests: Array<Record<string, unknown>> = []
    class StudioSocket {
      static OPEN = 1
      static CONNECTING = 0
      readyState = 0
      onopen: (() => void) | null = null
      onmessage: ((event: { data: string }) => void) | null = null
      onerror: (() => void) | null = null
      onclose: (() => void) | null = null

      constructor(_url: string) {
        queueMicrotask(() => {
          this.readyState = 1
          this.onopen?.()
        })
      }

      send(raw: string) {
        requests.push(JSON.parse(raw) as Record<string, unknown>)
        const emit = (event: Record<string, unknown>) =>
          this.onmessage?.({ data: JSON.stringify(event) })
        queueMicrotask(() => {
          emit({
            type: 'RUN_STARTED',
            threadId: 'session-1',
            runId: requests[0].runId,
            metadata: { studioRunId: 'studio-run-1' },
          })
          emit({
            type: 'TEXT_MESSAGE_START',
            messageId: 'reply-1',
            role: 'assistant',
            sequence: 1,
          })
          emit({
            type: 'TEXT_MESSAGE_CONTENT',
            messageId: 'reply-1',
            delta: '分镜已经生成',
            sequence: 2,
          })
          emit({ type: 'TEXT_MESSAGE_END', messageId: 'reply-1', sequence: 3 })
          emit({
            type: 'RUN_FINISHED',
            threadId: 'session-1',
            runId: requests[0].runId,
            sequence: 4,
            outcome: { type: 'success' },
          })
        })
      }

      close() {
        this.readyState = 3
      }
    }
    vi.stubGlobal('WebSocket', StudioSocket)
    try {
      const screen = await renderStudioChat({
        permissionMode: 'full_access',
        selectedSkillIds: ['storyboard-skill'],
        selectedAssets: [{ assetId: 'asset-1', assetVersionId: 'version-1' }],
      })
      await screen.getByRole('textbox', { name: '输入消息' }).fill('请把故事做成分镜')
      await screen.getByRole('button', { name: '发送消息' }).click()
      await expect.element(screen.getByText('分镜已经生成')).toBeVisible()
      expect(requests).toHaveLength(1)
      expect(requests[0]).toMatchObject({
        threadId: 'session-1',
        forwardedProps: {
          runConfig: {
            modelConfigId: 'model-1',
            permissionMode: 'full_access',
            selectedSkillIds: [],
            selectedAssets: [],
          },
        },
      })
      expect(JSON.stringify(requests[0].messages)).toContain('请把故事做成分镜')
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('does not offer an image-only model as an Agent model', async () => {
    const screen = await renderStudioChat({
      models: [
        {
          ...studioModel('image-only', '绘图模型'),
          capabilities: {
            image_output: true,
            streaming: false,
            tools: false,
            vision: false,
          },
        },
      ],
      modelConfigId: 'image-only',
    })
    await expect.element(screen.getByText('先在 AI 设置中添加并启用模型')).toBeVisible()
    await expect.element(screen.getByRole('textbox', { name: '输入消息' })).toHaveAttribute('contenteditable', 'false')
  })

  it('shows replayed reasoning before any assistant text after returning to a running chat', async () => {
    class ReplaySocket {
      readyState = 0
      onopen: (() => void) | null = null
      onmessage: ((event: { data: string }) => void) | null = null
      onerror: (() => void) | null = null
      onclose: (() => void) | null = null

      constructor(_url: string) {
        queueMicrotask(() => {
          this.readyState = 1
          this.onopen?.()
        })
      }

      send(raw: string) {
        const request = JSON.parse(raw) as { attachRunId?: string }
        if (request.attachRunId !== 'run-reasoning')
          throw new Error('未续接运行中的会话')
        for (const event of [
          { type: 'RUN_STARTED', metadata: { studioRunId: 'run-reasoning' } },
          {
            type: 'REASONING_MESSAGE_START',
            messageId: 'reason-1',
            sequence: 1,
          },
          {
            type: 'REASONING_MESSAGE_CONTENT',
            messageId: 'reason-1',
            delta: '仍在分析问题',
            sequence: 2,
          },
        ]) {
          this.onmessage?.({ data: JSON.stringify(event) })
        }
      }

      close() {
        this.readyState = 3
      }
    }
    vi.stubGlobal('WebSocket', ReplaySocket)
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const chat = () => (
      <QueryClientProvider client={client}>
        <StudioChat
          assets={[]}
          latestRun={{
            created_at: '2026-02-12T10:00:00Z',
            id: 'run-reasoning',
            session_id: 'session-reasoning',
            status: 'running',
            trigger_message_id: 'user-1',
          }}
          messages={[]}
          models={[]}
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
          sessionId='session-reasoning'
          skills={[]}
          transcript={{
            events: [],
            messages: [{ content: '测试问题', id: 'user-1', role: 'user' }],
          }}
        />
      </QueryClientProvider>
    )
    try {
      const screen = await render(chat())
      await expect
        .element(screen.getByRole('button', { name: '正在思考' }))
        .toBeVisible()
      await expect.element(screen.getByText('仍在分析问题')).toBeVisible()
    } finally {
      vi.unstubAllGlobals()
    }
  })

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
    const content = screen.getByRole('heading', { name: '创作大纲' }).element().closest('.is-assistant')?.firstElementChild as HTMLElement
    const conversationContent = content.closest('.max-w-3xl') as HTMLElement
    expect(parseFloat(getComputedStyle(conversationContent).rowGap)).toBeLessThanOrEqual(20)
    expect(parseFloat(getComputedStyle(content).rowGap)).toBeLessThanOrEqual(16)
    expect(content.getBoundingClientRect().width).toBe(content.parentElement!.getBoundingClientRect().width)
    await expect
      .element(screen.getByRole('textbox', { name: '输入消息' }))
      .toBeVisible()
  })

  it('keeps a tool call at the full message width when closed and open', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [
          { id: 'user-1', role: 'user', content: '生成大纲' },
          {
            id: 'assistant-1', role: 'assistant', content: '',
            toolCalls: [{ id: 'call-1', type: 'function', function: { name: 'make_outline', arguments: '{}' } }],
          },
          { id: 'tool-1', role: 'tool', toolCallId: 'call-1', content: '完成' },
          { id: 'assistant-2', role: 'assistant', content: '大纲已生成' },
        ],
      },
    })
    await expect.element(screen.getByText('make_outline')).toBeVisible()
    const trigger = screen.getByText('make_outline').element().closest('button')!
    const tool = trigger.parentElement as HTMLElement
    const message = tool.closest('.is-assistant') as HTMLElement
    expect(Math.abs(tool.getBoundingClientRect().width - message.getBoundingClientRect().width)).toBeLessThanOrEqual(1)
    expect(getComputedStyle(tool).marginBottom).toBe('0px')

    await trigger.click()
    expect(Math.abs(tool.getBoundingClientRect().width - message.getBoundingClientRect().width)).toBeLessThanOrEqual(1)
  })

  it('keeps a short sent message at its text width', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [{ id: 'user-1', role: 'user', content: '你好' }],
      },
    })
    await expect.element(screen.getByText('你好')).toBeVisible()
    const message = screen.getByText('你好').element().closest('.is-user') as HTMLElement
    const content = message.firstElementChild as HTMLElement
    expect(content.getBoundingClientRect().width).toBeLessThan(message.getBoundingClientRect().width / 2)
    expect(Math.abs(content.getBoundingClientRect().right - message.getBoundingClientRect().right)).toBeLessThanOrEqual(1)
  })

  it('keeps adjacent reply paragraphs close together', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [{ id: 'assistant-1', role: 'assistant', content: '第一段\n\n第二段' }],
      },
    })
    await expect.element(screen.getByText('第二段')).toBeVisible()
    const first = screen.getByText('第一段').element()
    const second = screen.getByText('第二段').element()
    expect(second.getBoundingClientRect().top - first.getBoundingClientRect().bottom).toBeLessThanOrEqual(8)
  })

  it('keeps a reply heading close to the preceding paragraph', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [{ id: 'assistant-1', role: 'assistant', content: '第一段\n\n## 标题\n\n第二段' }],
      },
    })
    await expect.element(screen.getByRole('heading', { name: '标题' })).toBeVisible()
    const paragraph = screen.getByText('第一段').element()
    const heading = screen.getByRole('heading', { name: '标题' }).element()
    expect(heading.getBoundingClientRect().top - paragraph.getBoundingClientRect().bottom).toBeLessThanOrEqual(12)
  })

  it('copies sent and returned message text', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [
          { id: 'user-1', role: 'user', content: '请写一个大纲' },
          { id: 'assistant-1', role: 'assistant', content: '这是大纲。' },
        ],
      },
    })
    await expect.element(screen.getByText('这是大纲。')).toBeVisible()
    const pasteTarget = document.createElement('textarea')
    document.body.append(pasteTarget)
    await screen.getByRole('button', { name: '复制发送消息' }).click()
    pasteTarget.focus()
    await userEvent.paste()
    expect(pasteTarget.value).toBe('请写一个大纲')
    await screen.getByRole('button', { name: '复制返回消息' }).click()
    pasteTarget.value = ''
    pasteTarget.focus()
    await userEvent.paste()
    expect(pasteTarget.value).toBe('这是大纲。')
    pasteTarget.remove()
  })

  it('copies one complete assistant response across tool calls', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [
          { id: 'user-1', role: 'user', content: '创建资产' },
          { id: 'assistant-1', role: 'assistant', content: '正在整理内容。' },
          {
            id: 'assistant-2', role: 'assistant', content: '',
            toolCalls: [{ id: 'call-1', type: 'function', function: { name: 'create_text_asset', arguments: '{}' } }],
          },
          { id: 'tool-1', role: 'tool', toolCallId: 'call-1', content: '创建完成' },
          { id: 'assistant-3', role: 'assistant', content: '资产已创建。' },
          { id: 'user-2', role: 'user', content: '下一步' },
          { id: 'assistant-4', role: 'assistant', content: '可以继续编辑。' },
        ],
      },
    })
    await expect.element(screen.getByText('可以继续编辑。')).toBeVisible()
    const buttons = screen.getByRole('button', { name: '复制返回消息' }).all()
    expect(buttons).toHaveLength(2)

    const pasteTarget = document.createElement('textarea')
    document.body.append(pasteTarget)
    await buttons[0].click()
    pasteTarget.focus()
    await userEvent.paste()
    expect(pasteTarget.value).toBe('正在整理内容。\n\n资产已创建。')

    await buttons[1].click()
    pasteTarget.value = ''
    pasteTarget.focus()
    await userEvent.paste()
    expect(pasteTarget.value).toBe('可以继续编辑。')
    pasteTarget.remove()
  })

  it('shows a pending confirmation in the conversation', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioChat
          assets={[]}
          latestRun={{
            created_at: '2026-02-12T10:00:00Z',
            id: 'run-1',
            session_id: 'session-1',
            status: 'waiting_approval',
            trigger_message_id: 'user-1',
          }}
          messages={[]}
          models={[]}
          onAssetChange={() => {}}
          onImportLibraryAsset={async () => {
            throw new Error('不应导入资产')
          }}
          onModelChange={() => {}}
          onPermissionChange={() => {}}
          onSkillChange={() => {}}
          pendingApprovals={[
            {
              id: 'approval-1',
              reason: 'tool_approval',
              message: '创建资产「大纲.md」',
            },
          ]}
          permissionMode='request_approval'
          selectedAssets={[]}
          selectedSkillIds={[]}
          sessionId='session-1'
          skills={[]}
        />
      </QueryClientProvider>
    )

    await expect.element(screen.getByText('创建资产「大纲.md」')).toBeVisible()
    await expect
      .element(screen.getByRole('status'))
      .toHaveTextContent('等待你批准操作')
    const composer = document.querySelector(
      "[data-slot='studio-composer']"
    ) as HTMLElement
    expect(composer.className).not.toContain('invisible')
    expect(screen.getByRole('log').element().contains(screen.getByRole('alert').element())).toBe(true)
    await expect.element(screen.getByRole('button', { name: /^批准$/ })).toBeDisabled()
  })

  it('keeps the composer visible while a pending confirmation loads', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    await render(
      <QueryClientProvider client={client}>
        <StudioChat
          assets={[]}
          latestRun={{
            created_at: '2026-02-12T10:00:00Z',
            id: 'run-1',
            session_id: 'session-1',
            status: 'waiting_approval',
            trigger_message_id: 'user-1',
          }}
          messages={[]}
          models={[]}
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
        />
      </QueryClientProvider>
    )

    const composer = document.querySelector(
      "[data-slot='studio-composer']"
    ) as HTMLElement
    expect(composer.className).not.toContain('invisible')
    expect(document.querySelector('[role="alert"]')).toBeNull()
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
    expect(getComputedStyle(group).borderTopLeftRadius).toBe('10px')
  })

  it('groups the model switcher with the send button and keeps Skills and assets as icon buttons', async () => {
    const onModelChange = vi.fn()
    const screen = await renderStudioChat({
      onModelChange,
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
    expect(inputRadius).toBe('10px')

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
      .getByRole('button', { name: '选择技能' })
      .element()
    const assetButton = screen
      .getByRole('button', { name: '选择资产' })
      .element()
    const permissionButton = screen
      .getByRole('button', { name: 'Agent 操作权限：请求批准' })
      .element()
    const leftGroup = footer.firstElementChild as HTMLElement
    expect(leftGroup.children[1]).toBe(skillButton)
    expect(leftGroup.children[2]).toBe(assetButton)
    expect(leftGroup.children[4]).toBe(permissionButton)
    for (const button of [skillButton, assetButton]) {
      expect(button.textContent).toBe('')
      expect(getComputedStyle(button).borderTopLeftRadius).toBe(inputRadius)
    }
    for (const button of [modelButton, permissionButton]) {
      expect(getComputedStyle(button).fontWeight).toBe('400')
    }

    await screen.getByRole('button', { name: '选择技能' }).hover()
    await expect
      .element(screen.getByRole('tooltip'))
      .toHaveTextContent('技能')

    await screen.getByRole('button', { name: 'Pixoma Chat' }).click()
    await expect.element(screen.getByRole('dialog')).toBeVisible()
    await screen.getByText('Pixoma Pro').click()
    expect(onModelChange).toHaveBeenCalledWith('model-2')
  })

  it('shows turn navigation with a preview for each user question', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [
          { id: 'user-1', role: 'user', content: '整理项目需求' },
          { id: 'assistant-1', role: 'assistant', content: '需求已经整理。' },
          { id: 'user-2', role: 'user', content: '列出开发任务' },
          { id: 'assistant-2', role: 'assistant', content: '开发任务已经列出。' },
        ],
      },
    })

    const chat = document.querySelector<HTMLElement>('.studio-chat-root')!
    chat.style.width = '1200px'
    chat.style.height = '700px'
    const scroll = chat.querySelector<HTMLElement>('.studio-scrollbar')!
    expect(getComputedStyle(scroll).scrollbarWidth).toBe('none')
    expect(getComputedStyle(scroll).scrollbarGutter).toBe('auto')
    const navigation = screen.getByRole('navigation', { name: '轮次导航' })
    const navigationSlot = navigation.element().parentElement!
    const firstTurn = navigation.getByRole('button', { name: '跳转到第 1 轮' })
    await expect.element(firstTurn).toBeVisible()
    await expect.element(navigation.getByRole('button', { name: '跳转到第 2 轮' })).toBeVisible()
    expect(getComputedStyle(firstTurn.element(), '::before').width).toBe('10px')

    await firstTurn.hover()
    await expect.element(screen.getByRole('tooltip')).toHaveTextContent('整理项目需求')
    await expect.element(screen.getByRole('tooltip')).toHaveTextContent('需求已经整理。')
    expect(firstTurn.element().getBoundingClientRect().right - 10 - screen.getByRole('tooltip').element().getBoundingClientRect().right).toBe(8)

    const activeTurn = navigation.getByRole('button', { name: '跳转到第 2 轮' })
    await activeTurn.hover()
    const preview = screen.getByRole('tooltip')
    await expect.element(preview).toHaveTextContent('列出开发任务')
    expect(activeTurn.element().getBoundingClientRect().right - 10 - preview.element().getBoundingClientRect().right).toBe(8)

    chat.style.width = '920px'
    await expect.poll(() => getComputedStyle(navigationSlot).display).toBe('none')
    expect(getComputedStyle(scroll).scrollbarWidth).toBe('thin')

    chat.style.width = '921px'
    await expect.element(firstTurn).toBeVisible()
    expect(getComputedStyle(scroll).scrollbarWidth).toBe('none')
  })

  it('shows turn navigation after the first user question', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [
          { id: 'user-1', role: 'user', content: '整理项目需求' },
          { id: 'assistant-1', role: 'assistant', content: '需求已经整理。' },
        ],
      },
    })

    const chat = document.querySelector<HTMLElement>('.studio-chat-root')!
    chat.style.width = '1200px'
    chat.style.height = '700px'
    const navigation = screen.getByRole('navigation', { name: '轮次导航' })
    await expect.element(navigation.getByRole('button', { name: '跳转到第 1 轮' })).toBeVisible()
  })

  it('jumps from the latest reply to the selected turn', async () => {
    const screen = await renderStudioChat({
      transcript: {
        events: [],
        messages: [
          { id: 'user-1', role: 'user', content: '整理项目需求' },
          { id: 'assistant-1', role: 'assistant', content: '需求已经整理。' },
          { id: 'user-2', role: 'user', content: '列出开发任务' },
          { id: 'assistant-2', role: 'assistant', content: '开发任务已经列出。' },
        ],
      },
    })
    const chat = document.querySelector<HTMLElement>('.studio-chat-root')!
    chat.style.width = '1200px'
    chat.style.height = '320px'
    const scroll = chat.querySelector<HTMLElement>('.studio-scrollbar')!
    const firstTurn = screen.getByRole('navigation', { name: '轮次导航' })
      .getByRole('button', { name: '跳转到第 1 轮' })

    await expect.poll(() => scroll.scrollTop).toBeGreaterThan(0)
    const previousTop = scroll.scrollTop
    await firstTurn.click()

    await expect.poll(() => scroll.scrollTop).toBeLessThan(previousTop)
    await expect.element(firstTurn).toHaveAttribute('aria-current', 'true')
  })

  it('scrolls a long turn rail independently from the conversation', async () => {
    const messages = Array.from({ length: 100 }, (_, index) => [
      { id: `user-${index}`, role: 'user' as const, content: `第 ${index + 1} 个问题` },
      { id: `assistant-${index}`, role: 'assistant' as const, content: `第 ${index + 1} 个回答` },
    ]).flat()
    const screen = await renderStudioChat({ transcript: { events: [], messages } })
    const chat = document.querySelector<HTMLElement>('.studio-chat-root')!
    chat.style.width = '1200px'
    chat.style.height = '700px'
    const navigation = screen.getByRole('navigation', { name: '轮次导航' })
    const rail = navigation.element().querySelector<HTMLElement>('[data-slot="studio-turn-rail-scroll"]')!
    const conversation = chat.querySelector<HTMLElement>('.studio-scrollbar')!

    await expect.poll(() => rail.scrollHeight > rail.clientHeight).toBe(true)
    expect(navigation.element().querySelectorAll('button').length).toBeLessThan(100)
    const conversationTop = conversation.scrollTop
    rail.scrollTop = 80

    await expect.poll(() => rail.scrollTop).toBe(80)
    expect(conversation.scrollTop).toBe(conversationTop)
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

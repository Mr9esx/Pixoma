import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { StudioActionDock, StudioChat } from './studio-chat'

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
  })

  it('peeks the pending approvals above the composer and reveals the choices on demand', async () => {
    const responded: Array<[string, boolean]> = []
    function Harness() {
      const [open, setOpen] = useState(false)
      return (
        <StudioActionDock
          actions={[{ id: 'interrupt-1', message: '需要写入 Session 资产' }]}
          open={open}
          onToggle={() => setOpen((current) => !current)}
          onRespond={(id, approved) => responded.push([id, approved])}
        />
      )
    }
    const screen = await render(<Harness />)

    const dock = screen.getByRole('region', { name: '操作区' }).element()
    const peek = screen.getByRole('button', { name: /需要你的批准 · 1 项/ })
    expect(getComputedStyle(dock).height).toBe('44px')
    expect(getComputedStyle(dock).overflow).toBe('hidden')
    expect(peek.element().getAttribute('aria-expanded')).toBe('false')
    await expect
      .element(screen.getByRole('button', { name: /^批准$/ }))
      .not.toBeInTheDocument()

    await peek.click()
    expect(peek.element().getAttribute('aria-expanded')).toBe('true')
    expect(parseFloat(getComputedStyle(dock).height)).toBeGreaterThan(44)
    expect(screen.getByText('需要写入 Session 资产')).toBeVisible()
    await screen.getByRole('button', { name: /^批准$/ }).click()
    expect(responded).toEqual([['interrupt-1', true]])

    await screen.getByRole('button', { name: /^拒绝$/ }).click()
    expect(responded).toEqual([
      ['interrupt-1', true],
      ['interrupt-1', false],
    ])
  })
})

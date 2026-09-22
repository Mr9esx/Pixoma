import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { StudioChat } from './studio-chat'

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
})

import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import type { StudioModel } from '@/lib/api/studio'
import { StudioSettings, type SettingSection } from './studio-settings'

const model: StudioModel = {
  id: 'model-1',
  name: '故事模型',
  protocol: 'openai_chat_compatible',
  base_url: 'https://example.com/v1/chat/completions',
  model: 'story-model',
  has_api_key: true,
  enabled: true,
  agent_enabled: true,
  default: true,
  limits: {
    context_window_tokens: 128000,
    max_input_tokens: 120000,
    max_output_tokens: 4096,
  },
  thinking: { enabled: false },
  capabilities: {
    tools: true,
    vision: false,
    image_output: false,
    streaming: true,
  },
}

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  listStudioModels: vi.fn(async () => [model]),
  listStudioSkills: vi.fn(async () => []),
  listStudioConnectors: vi.fn(async () => []),
  listStudioAgentWorkflows: vi.fn(async () => []),
}))

function TestSettings() {
  const [section, setSection] = useState<SettingSection>('models')
  return <StudioSettings section={section} onSectionChange={setSection} />
}

describe('StudioSettings', () => {
  it('does not allow Agent access when tool calling is unsupported', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <TestSettings />
      </QueryClientProvider>
    )
    await screen.getByRole('button', { name: '添加模型' }).click()
    await screen.getByRole('switch', { name: '支持工具调用' }).click()
    await expect
      .element(screen.getByRole('switch', { name: '允许 Agent 使用' }))
      .toBeDisabled()
    await expect
      .element(screen.getByRole('switch', { name: '允许 Agent 使用' }))
      .not.toBeChecked()
  })

  it('opens a fresh model form after canceling an edit', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <TestSettings />
      </QueryClientProvider>
    )
    await screen.getByRole('button', { name: '编辑' }).click()
    await expect
      .element(screen.getByRole('textbox', { name: '名称' }))
      .toHaveValue('故事模型')
    await screen.getByRole('button', { name: '取消' }).click()
    await screen.getByRole('button', { name: '添加模型' }).click()
    await expect
      .element(screen.getByRole('textbox', { name: '名称' }))
      .toHaveValue('')
  })
})

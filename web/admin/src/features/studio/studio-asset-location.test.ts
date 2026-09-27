import { createElement, useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import { StudioChat } from './studio-chat'

const transcript = {
  events: [],
  messages: Array.from({ length: 30 }, (_, index) => [
    {
      id: `user-${index}`,
      runId: `run-${index}`,
      role: 'user' as const,
      content: `第 ${index + 1} 个问题：${'生成内容。'.repeat(12)}`,
    },
    {
      id: `assistant-${index}`,
      runId: `run-${index}`,
      role: 'assistant' as const,
      content: `第 ${index + 1} 个回答：${'内容已生成。'.repeat(12)}`,
    },
  ]).flat(),
}

function LocationHarness() {
  const [locateMessage, setLocateMessage] = useState<{
    id: string
    request: number
  }>()
  return createElement(
    'div',
    { className: 'flex h-[420px] w-[800px] flex-col' },
    createElement(
      'button',
      {
        type: 'button',
        onClick: () =>
          setLocateMessage((current) => ({
            id: 'user-0',
            request: (current?.request ?? 0) + 1,
          })),
      },
      '定位生成对话'
    ),
    createElement(StudioChat, {
      sessionId: 'session-1',
      messages: [],
      transcript,
      models: [],
      permissionMode: 'request_approval',
      skills: [],
      assets: [],
      selectedSkillIds: [],
      selectedAssets: [],
      onModelChange: () => {},
      onPermissionChange: () => {},
      onSkillChange: () => {},
      onAssetChange: () => {},
      onImportLibraryAsset: async () => {
        throw new Error('未触发导入资产')
      },
      locateMessage,
    })
  )
}

describe('会话资产来源定位', () => {
  it('把对话滚动到生成资产的轮次', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { staleTime: Infinity } },
    })
    const screen = await render(
      createElement(
        QueryClientProvider,
        { client },
        createElement(LocationHarness)
      )
    )
    const scroll =
      screen.container.querySelector<HTMLElement>('.studio-scrollbar')!
    await expect
      .poll(() => scroll.scrollHeight > scroll.clientHeight)
      .toBe(true)
    scroll.scrollTop = scroll.scrollHeight
    const previousTop = scroll.scrollTop
    await screen.getByRole('button', { name: '定位生成对话' }).click()
    await expect.poll(() => scroll.scrollTop).toBeLessThan(previousTop)
    const anchor = scroll.querySelector<HTMLElement>(
      '[data-studio-turn-id="user-0"]'
    )!
    await expect
      .poll(() => anchor.getBoundingClientRect().top, { timeout: 4000 })
      .toBeGreaterThanOrEqual(scroll.getBoundingClientRect().top)
  })
})

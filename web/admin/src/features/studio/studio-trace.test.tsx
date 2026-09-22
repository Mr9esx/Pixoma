import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { StudioTrace } from './studio-trace'

const fixtures = vi.hoisted(() => ({
  runs: [
    {
      id: 'run-1',
      session_id: 'session-1',
      trigger_message_id: 'message-1',
      status: 'succeeded' as const,
      created_at: '2026-09-22T10:00:00.000Z',
      updated_at: '2026-09-22T10:00:03.000Z',
      started_at: '2026-09-22T10:00:00.000Z',
      completed_at: '2026-09-22T10:00:03.000Z',
    },
  ],
  events: [
    {
      id: 'event-1',
      run_id: 'run-1',
      sequence: 1,
      type: 'TEXT_MESSAGE_START',
      payload: { message_id: 'assistant-1', role: 'assistant' },
      created_at: '2026-09-22T10:00:00.000Z',
    },
    {
      id: 'event-2',
      run_id: 'run-1',
      sequence: 2,
      type: 'TEXT_MESSAGE_CONTENT',
      payload: { message_id: 'assistant-1', delta: '第一段' },
      created_at: '2026-09-22T10:00:01.000Z',
    },
    {
      id: 'event-3',
      run_id: 'run-1',
      sequence: 3,
      type: 'TEXT_MESSAGE_CONTENT',
      payload: { message_id: 'assistant-1', delta: '第二段' },
      created_at: '2026-09-22T10:00:02.000Z',
    },
    {
      id: 'event-4',
      run_id: 'run-1',
      sequence: 4,
      type: 'TEXT_MESSAGE_END',
      payload: { message_id: 'assistant-1', content: '第一段第二段' },
      created_at: '2026-09-22T10:00:03.000Z',
    },
    {
      id: 'event-5',
      run_id: 'run-1',
      sequence: 5,
      type: 'TOOL_CALL_START',
      payload: { tool_call_id: 'tool-1', tool_name: '搜索资料' },
      created_at: '2026-09-22T10:00:03.000Z',
    },
    {
      id: 'event-6',
      run_id: 'run-1',
      sequence: 6,
      type: 'TOOL_CALL_RESULT',
      payload: { tool_call_id: 'tool-1', content: '找到 3 条资料', is_error: false },
      created_at: '2026-09-22T10:00:03.200Z',
    },
    {
      id: 'event-7',
      run_id: 'run-1',
      sequence: 7,
      type: 'TOOL_CALL_END',
      payload: { tool_call_id: 'tool-1', tool_name: '搜索资料' },
      created_at: '2026-09-22T10:00:03.400Z',
    },
    {
      id: 'event-8',
      run_id: 'run-1',
      sequence: 8,
      type: 'ASSET_CREATED',
      payload: { asset_id: 'asset-1', name: '资料摘录.md' },
      created_at: '2026-09-22T10:00:03.500Z',
    },
  ],
}))

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  listStudioSessionRuns: vi.fn(() => Promise.resolve(fixtures.runs)),
  listStudioRunEvents: vi.fn(() => Promise.resolve(fixtures.events)),
}))

describe('StudioTrace', () => {
  it('groups streamed output into one trajectory record with an overview timeline', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioTrace sessionId='session-1' />
      </QueryClientProvider>
    )

    await expect.element(screen.getByText('轨迹总览')).toBeVisible()
    await expect.element(screen.getByText('第一段第二段')).toBeVisible()
    await expect.element(screen.getByText('搜索资料')).toBeVisible()
    await expect.element(screen.getByText('创建资产')).toBeVisible()
  })
})

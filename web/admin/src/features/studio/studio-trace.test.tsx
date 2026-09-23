import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { StudioTrace } from './studio-trace'

const fixtures = vi.hoisted(() => ({
  pages: [
    {
      runs: [
        {
          run: {
            id: 'run-2',
            session_id: 'session-1',
            status: 'succeeded',
            trigger_message_id: 'user-2',
            created_at: '2026-09-22T10:10:00Z',
            started_at: '2026-09-22T10:10:00Z',
            completed_at: '2026-09-22T10:10:02Z',
          },
          records: [
            {
              id: 'model-2',
              run_id: 'run-2',
              kind: 'model',
              title: '模型 B',
              status: 'done',
              step: 1,
              attempt: 1,
              started_at: '2026-09-22T10:10:00Z',
            },
            {
              id: 'tool-2',
              run_id: 'run-2',
              kind: 'tool',
              title: '搜索资料',
              status: 'done',
              step: 1,
              started_at: '2026-09-22T10:10:01Z',
            },
          ],
        },
        {
          run: {
            id: 'run-1',
            session_id: 'session-1',
            status: 'succeeded',
            trigger_message_id: 'user-1',
            created_at: '2026-09-22T10:00:00Z',
            started_at: '2026-09-22T10:00:00Z',
            completed_at: '2026-09-22T10:00:02Z',
          },
          records: [
            {
              id: 'model-1',
              run_id: 'run-1',
              kind: 'model',
              title: '模型 A',
              status: 'done',
              step: 1,
              attempt: 1,
              started_at: '2026-09-22T10:00:00Z',
            },
          ],
        },
      ],
      next_cursor: '',
      has_more: false,
      total_runs: 2,
    },
  ],
}))

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  getStudioSessionTrajectory: vi.fn(() => Promise.resolve(fixtures.pages[0])),
  getStudioTrajectoryRecord: vi.fn(
    (_session: string, run: string, record: string) =>
      Promise.resolve({
        record: {
          id: record,
          run_id: run,
          kind: 'model',
          title: '模型 A',
          status: 'done',
          started_at: '2026-09-22T10:00:00Z',
        },
        overview: { model: 'test-model' },
        input: { messages: ['实际请求'] },
        output: { choices: ['实际响应'] },
        raw: { messages: ['实际请求'] },
        usage: { input_tokens: 10, output_tokens: 0, source: 'provider' },
        timing: { ttft_ms: 100 },
      })
  ),
}))

describe('StudioTrace', () => {
  it('keeps a call summary when calls are collapsed and can expand them', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioTrace sessionId='session-1' />
      </QueryClientProvider>
    )
    await expect.element(screen.getByRole('row', { name: /工具 搜索资料/ })).toBeVisible()
    await screen.getByRole('button', { name: '收起所有调用' }).click()
    await expect.element(screen.getByRole('row', { name: /工具 搜索资料/ })).not.toBeInTheDocument()
    await expect.element(screen.getByRole('button', { name: '展开 1 次工具调用' })).toBeVisible()
    await screen.getByRole('button', { name: '展开 1 次工具调用' }).click()
    await expect.element(screen.getByRole('row', { name: /工具 搜索资料/ })).toBeVisible()
  })

  it('shows the session across turns and lazy record detail', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioTrace sessionId='session-1' />
      </QueryClientProvider>
    )
    await expect
      .element(screen.getByRole('toolbar', { name: '轨迹工具栏' }))
      .toBeVisible()
    await expect.element(screen.getByLabelText('轨迹时间线')).toBeVisible()
    await expect
      .element(screen.getByRole('table', { name: '轨迹账本' }))
      .toBeVisible()
    await expect.element(screen.getByText('第 1 轮')).toBeVisible()
    await expect.element(screen.getByText('第 2 轮')).toBeVisible()
    await expect
      .element(screen.getByRole('button', { name: '选择模型 A' }))
      .toBeVisible()
    await screen.getByRole('row', { name: /● 模型 模型 A/ }).click()
    await expect
      .element(screen.getByRole('complementary', { name: '事件详情' }))
      .toBeVisible()
    await screen.getByRole('tab', { name: '输入' }).click()
    await expect.element(screen.getByText(/实际请求/)).toBeVisible()
    await screen.getByRole('tab', { name: '用量' }).click()
    await expect.element(screen.getByText(/input_tokens/)).toBeVisible()
    await screen.getByRole('button', { name: '关闭详情' }).click()
    await expect
      .element(screen.getByRole('complementary', { name: '事件详情' }))
      .not.toBeInTheDocument()
    await screen.getByRole('button', { name: '使用实际时长' }).click()
    await expect.element(screen.getByRole('button', { name: '使用等宽操作' })).toBeVisible()
    await screen.getByRole('button', { name: '收起所有轮次' }).click()
    await expect.element(screen.getByRole('row', { name: /● 模型 模型 A/ })).not.toBeInTheDocument()
  })
})

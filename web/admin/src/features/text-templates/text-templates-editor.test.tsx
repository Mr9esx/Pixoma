import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TextTemplatesEditor } from './text-templates-editor'
import type { TextTemplate } from '@/lib/api/text-templates'

const templates: TextTemplate[] = [
  {
    key: 'confirm_run',
    group: 'workflow',
    description: '确认执行前的确认文案',
    default: '输入完成，确认执行？',
    value: '',
  },
  {
    key: 'task_failed',
    group: 'notifications',
    description: '任务失败通知',
    default: '任务失败',
    value: '',
  },
  {
    key: 'welcome',
    group: 'platform',
    description: '主菜单标题',
    default: '欢迎',
    value: '',
  },
  {
    key: 'help',
    group: 'commands',
    description: '/help 帮助文案',
    default: '帮助',
    value: '',
  },
]

vi.mock('@/lib/api/text-templates', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/text-templates')>()),
  listTextTemplates: vi.fn(() => Promise.resolve(templates)),
}))

describe('TextTemplatesEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders all scenario groups in one page', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <TextTemplatesEditor channelId="" />
      </QueryClientProvider>
    )

    for (const heading of ['工作流阶段', '任务通知', '平台与会话', '公开命令']) {
      await expect.element(screen.getByText(heading)).toBeVisible()
    }
    await expect.element(screen.getByText('confirm_run')).toBeVisible()
    expect(screen.container.querySelector('[data-menu-editor]')).toBeNull()
  })
})

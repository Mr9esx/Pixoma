import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import { getStudioSession, type StudioSessionDetail } from '@/lib/api/studio'
import { StudioWorkspace } from './studio-workspace'

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  getStudioSession: vi.fn(),
  listStudioSessions: vi.fn(),
  listStudioModels: vi.fn(async () => []),
  listStudioSkills: vi.fn(async () => []),
}))

vi.mock('./studio-sidebar', () => ({
  StudioSidebar: ({
    onSelectSession,
  }: {
    onSelectSession: (id: string) => void
  }) => <button onClick={() => onSelectSession('session-a')}>打开 A</button>,
}))

vi.mock('./studio-chat', () => ({
  StudioChat: ({
    sessionId,
    latestRun,
  }: {
    sessionId: string
    latestRun?: { status: string } | null
  }) => (
    <div data-testid='chat-state'>
      {sessionId}:{latestRun?.status ?? 'idle'}
    </div>
  ),
}))

vi.mock('./studio-flow', () => ({ StudioFlow: () => null }))
vi.mock('./studio-assets', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./studio-assets')>()),
  StudioAssets: () => null,
}))

const detail = (id: string, status?: 'running'): StudioSessionDetail => ({
  session: {
    id,
    title: id,
    permission_mode: 'request_approval',
    status: 'active',
    created_at: '2026-02-12T10:00:00Z',
    updated_at: '2026-02-12T10:00:00Z',
    ...(status
      ? {
          latest_run: {
            id: 'run-a',
            session_id: id,
            trigger_message_id: 'user-a',
            status,
            created_at: '2026-02-12T10:01:00Z',
            updated_at: '2026-02-12T10:01:00Z',
          },
        }
      : {}),
  },
  messages: [],
  transcript: { messages: [], events: [] },
  assets: [],
  flow: { nodes: [], edges: [] },
})

describe('StudioWorkspace', () => {
  it('recovers from a failed session list without leaving the chat skeleton indefinitely', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const session = detail('session-after-retry')
    const api = await import('@/lib/api/studio')
    let connected = false
    vi.mocked(api.listStudioSessions).mockImplementation(async () => {
      if (!connected) throw new Error('network unavailable')
      return [session.session]
    })
    vi.mocked(getStudioSession).mockResolvedValue(session)

    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioWorkspace />
      </QueryClientProvider>
    )
    await expect.element(screen.getByText('对话列表读取失败')).toBeVisible()
    connected = true
    await screen.getByRole('button', { name: '重试读取对话' }).click()
    await expect
      .element(screen.getByTestId('chat-state'))
      .toHaveTextContent('session-after-retry:idle')
  })

  it('lets a failed conversation read retry and restores the chat', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const session = detail('session-retry')
    const api = await import('@/lib/api/studio')
    vi.mocked(api.listStudioSessions).mockResolvedValue([session.session])
    vi.mocked(getStudioSession)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValue(session)

    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioWorkspace />
      </QueryClientProvider>
    )
    await expect.element(screen.getByText('对话读取失败')).toBeVisible()
    await screen.getByRole('button', { name: '重试读取' }).click()
    await expect
      .element(screen.getByTestId('chat-state'))
      .toHaveTextContent('session-retry:idle')
  })

  it('opens the Flow and Session assets workbench on a narrow screen', async () => {
    const originalViewport = {
      width: window.innerWidth,
      height: window.innerHeight,
    }
    await page.viewport(390, 844)
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const session = detail('session-mobile')
    const api = await import('@/lib/api/studio')
    vi.mocked(api.listStudioSessions).mockResolvedValue([session.session])
    vi.mocked(getStudioSession).mockResolvedValue(session)
    try {
      const screen = await render(
        <QueryClientProvider client={client}>
          <StudioWorkspace />
        </QueryClientProvider>
      )
      await expect.element(screen.getByTestId('chat-state')).toBeVisible()
      await screen.getByRole('button', { name: '打开创作工作台' }).click()
      const workbench = screen.getByRole('dialog', { name: '创作工作台' })
      await expect.element(workbench).toBeVisible()
      await expect
        .element(workbench.getByRole('tab', { name: '资产路线' }))
        .toBeVisible()
      await workbench.getByRole('tab', { name: /Session 资产/ }).click()
      await expect
        .element(workbench.getByRole('tab', { name: /Session 资产/ }))
        .toHaveAttribute('aria-selected', 'true')
    } finally {
      await page.viewport(originalViewport.width, originalViewport.height)
    }
  })

  it('refreshes cached session detail before mounting a returned chat', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const staleA = detail('session-a')
    const freshA = detail('session-a', 'running')
    const sessionB = detail('session-b')
    const pendingA = Promise.withResolvers<StudioSessionDetail>()
    client.setQueryData(['studio', 'session', 'session-a'], staleA)
    vi.mocked(getStudioSession).mockImplementation(async (id) =>
      id === 'session-a' ? pendingA.promise : sessionB
    )
    const api = await import('@/lib/api/studio')
    vi.mocked(api.listStudioSessions).mockResolvedValue([
      sessionB.session,
      freshA.session,
    ])

    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioWorkspace />
      </QueryClientProvider>
    )
    await expect
      .element(screen.getByTestId('chat-state'))
      .toHaveTextContent('session-b:idle')
    await screen.getByRole('button', { name: '打开 Studio 菜单' }).click()
    const menu = screen.getByRole('dialog', { name: 'Studio 菜单' }).element()
    expect(
      document.getElementById(menu.getAttribute('aria-describedby') ?? '')
        ?.textContent
    ).toBe('切换对话、资产库和 AI 设置。')
    await screen.getByRole('button', { name: '打开 A' }).first().click()
    expect(
      document.querySelector('[data-testid="chat-state"]')?.textContent
    ).not.toBe('session-a:idle')

    pendingA.resolve(freshA)
    await expect
      .element(screen.getByTestId('chat-state'))
      .toHaveTextContent('session-a:running')
  })
})

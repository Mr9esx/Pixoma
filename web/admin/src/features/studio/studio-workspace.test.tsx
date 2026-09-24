import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { page } from 'vitest/browser'
import {
  createStudioSession,
  getStudioSession,
  type StudioSessionDetail,
} from '@/lib/api/studio'
import { StudioWorkspace } from './studio-workspace'

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  getStudioSession: vi.fn(),
  createStudioSession: vi.fn(),
  listStudioSessions: vi.fn(),
  listStudioModels: vi.fn(async () => []),
  listStudioSkills: vi.fn(async () => []),
}))

vi.mock('./studio-sidebar', () => ({
  StudioSidebar: ({
    onSelectSession,
    onNewSession,
    onViewChange,
  }: {
    onSelectSession: (id: string) => void
    onNewSession: () => void
    onViewChange: (view: 'chat' | 'library' | 'settings') => void
  }) => (
    <>
      <button onClick={() => onSelectSession('session-a')}>打开 A</button>
      <button onClick={onNewSession}>新建对话</button>
      <button onClick={() => onViewChange('library')}>打开资产库</button>
      <button onClick={() => onViewChange('settings')}>打开 AI 设置</button>
    </>
  ),
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
vi.mock('./studio-library', () => ({
  StudioLibrary: () => <div>资产库页面</div>,
}))
vi.mock('./studio-settings', () => ({
  StudioSettings: () => <div>AI 设置页面</div>,
}))
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
  beforeEach(() => vi.clearAllMocks())

  it('offers a retry when automatic conversation creation fails', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const session = detail('session-after-create-retry')
    const api = await import('@/lib/api/studio')
    vi.mocked(api.listStudioSessions).mockResolvedValue([])
    vi.mocked(getStudioSession).mockResolvedValue(session)
    vi.mocked(createStudioSession)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce(session.session)

    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioWorkspace />
      </QueryClientProvider>
    )
    await expect.element(screen.getByText('新建对话失败')).toBeVisible()
    await screen.getByRole('button', { name: '重试新建对话' }).click()
    await expect
      .element(screen.getByTestId('chat-state'))
      .toHaveTextContent('session-after-create-retry:idle')
    const requestIDs = vi
      .mocked(createStudioSession)
      .mock.calls.map(([requestID]) => requestID)
    expect(requestIDs).toHaveLength(2)
    expect(requestIDs[0]).toBe(requestIDs[1])
    expect(requestIDs[0]).toMatch(/^[0-9a-f-]{36}$/)
  })

  it('starts a fresh request after a lost create response is recovered by the session list', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const api = await import('@/lib/api/studio')
    vi.mocked(api.listStudioSessions).mockResolvedValue([])
    vi.mocked(getStudioSession).mockImplementation(async (id) => detail(id))
    vi.mocked(createStudioSession)
      .mockRejectedValueOnce(new Error('response lost'))
      .mockResolvedValueOnce(detail('fresh-session').session)

    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioWorkspace />
      </QueryClientProvider>
    )
    await expect.element(screen.getByText('新建对话失败')).toBeVisible()
    const lostRequestID = vi.mocked(createStudioSession).mock.calls[0][0]
    client.setQueryData(
      ['studio', 'sessions'],
      [detail('recovered-session').session]
    )
    await expect
      .element(screen.getByTestId('chat-state'))
      .toHaveTextContent('recovered-session:idle')

    await screen.getByRole('button', { name: '打开 Studio 菜单' }).click()
    await screen
      .getByRole('button', { name: '新建对话', exact: true })
      .first()
      .click()
    await expect
      .element(screen.getByTestId('chat-state'))
      .toHaveTextContent('fresh-session:idle')
    expect(vi.mocked(createStudioSession).mock.calls[1][0]).not.toBe(
      lostRequestID
    )
  })

  it.each([
    ['library', '打开资产库', '资产库页面'],
    ['settings', '打开 AI 设置', 'AI 设置页面'],
  ])(
    'shows a failed create and same-key retry from %s',
    async (_, openView, pageName) => {
      const originalViewport = {
        width: window.innerWidth,
        height: window.innerHeight,
      }
      await page.viewport(1440, 900)
      try {
        const client = new QueryClient({
          defaultOptions: { queries: { retry: false, refetchInterval: false } },
        })
        const session = detail('session-existing')
        const created = detail('session-after-retry')
        const api = await import('@/lib/api/studio')
        vi.mocked(api.listStudioSessions).mockResolvedValue([session.session])
        vi.mocked(getStudioSession).mockImplementation(async (id) => detail(id))
        vi.mocked(createStudioSession)
          .mockRejectedValueOnce(new Error('network unavailable'))
          .mockResolvedValueOnce(created.session)

        const screen = await render(
          <QueryClientProvider client={client}>
            <StudioWorkspace />
          </QueryClientProvider>
        )
        await screen.getByRole('button', { name: openView }).click()
        await expect.element(screen.getByText(pageName)).toBeVisible()
        await screen
          .getByRole('button', { name: '新建对话', exact: true })
          .click()
        await expect
          .element(screen.getByRole('alert'))
          .toHaveTextContent('新建对话失败')
        await screen.getByRole('button', { name: '重试新建对话' }).click()
        await expect
          .element(screen.getByTestId('chat-state'))
          .toHaveTextContent('session-after-retry:idle')
        const requestIDs = vi
          .mocked(createStudioSession)
          .mock.calls.map(([id]) => id)
        expect(requestIDs).toHaveLength(2)
        expect(requestIDs[0]).toBe(requestIDs[1])
      } finally {
        await page.viewport(originalViewport.width, originalViewport.height)
      }
    }
  )

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

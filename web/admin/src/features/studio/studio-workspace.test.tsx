import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import type { StudioAsset, StudioSessionDetail } from '@/lib/api/studio'
import { StudioLibrary } from './studio-library'

const asset: StudioAsset = {
  id: 'asset-1', session_id: 'session-1', name: '故事设定.md', kind: 'document', origin: 'agent',
  current_version: 1, saved_to_library: true, created_at: '2026-09-20T10:00:00Z', updated_at: '2026-09-20T10:00:00Z',
  versions: [{ id: 'version-1', version: 1, mime_type: 'text/markdown', size_bytes: 12, content_url: '/api/v1/studio/assets/asset-1/versions/version-1/content', created_at: '2026-09-20T10:00:00Z' }],
}

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  listStudioLibraryFolders: vi.fn(async () => []),
  listStudioLibraryAssets: vi.fn(async () => [asset]),
  getStudioSession: vi.fn(async () => ({ session: { id: 'session-1', title: '雨夜侦探漫画' } }) as StudioSessionDetail),
  getStudioTextAssetContent: vi.fn(async () => '# 雨夜侦探'),
}))

describe('StudioLibrary', () => {
  it('opens a saved asset with its content and source conversation', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const onOpenSession = vi.fn()
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioLibrary onOpenSession={onOpenSession} />
      </QueryClientProvider>,
    )
    await screen.getByRole('button', { name: '查看' }).click()
    await expect.element(screen.getByText('# 雨夜侦探')).toBeVisible()
    await expect.element(screen.getByText('雨夜侦探漫画')).toBeVisible()
    await screen.getByRole('button', { name: '打开来源对话' }).click()
    expect(onOpenSession).toHaveBeenCalledWith('session-1')
  })
})

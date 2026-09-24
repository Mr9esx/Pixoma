import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { page } from 'vitest/browser'
import { render } from 'vitest-browser-react'
import * as studioApi from '@/lib/api/studio'
import type { StudioAsset, StudioSessionDetail } from '@/lib/api/studio'
import { StudioLibrary } from './studio-library'

const asset: StudioAsset = {
  id: 'asset-1', session_id: 'session-1', name: '故事设定.md', kind: 'document', origin: 'agent',
  current_version: 1, saved_to_library: true, created_at: '2026-09-20T10:00:00Z', updated_at: '2026-09-20T10:00:00Z',
  versions: [{ id: 'version-1', version: 1, mime_type: 'text/markdown', size_bytes: 12, content_url: '/api/v1/studio/assets/asset-1/versions/version-1/content', created_at: '2026-09-20T10:00:00Z' }],
}

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  listStudioLibraryFolders: vi.fn(async () => [{ id: 'folder-1', name: '角色设定', parent_id: '', created_at: '', updated_at: '' }]),
  listStudioLibraryAssets: vi.fn(async () => [asset]),
  getStudioSession: vi.fn(async () => ({ session: { id: 'session-1', title: '雨夜侦探漫画' } }) as StudioSessionDetail),
  getStudioTextAssetContent: vi.fn(async () => '# 雨夜侦探'),
  uploadStudioAsset: vi.fn(async () => { throw new Error('网络已断开') }),
  moveStudioLibraryAsset: vi.fn(async () => undefined),
}))

describe('StudioLibrary', () => {
  it('moves a saved asset into a folder', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioLibrary onOpenSession={() => {}} />
      </QueryClientProvider>,
    )
    await screen.getByRole('button', { name: '查看' }).click()
    await screen.getByRole('button', { name: '移动到文件夹' }).click()
    await screen.getByRole('combobox', { name: '目标文件夹' }).click()
    await screen.getByRole('option', { name: '角色设定' }).click()
    await screen.getByRole('button', { name: '移动资产' }).click()
    expect(studioApi.moveStudioLibraryAsset).toHaveBeenCalledWith('asset-1', 'folder-1')
  })
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

  it('retries loading assets after a request fails', async () => {
    vi.mocked(studioApi.listStudioLibraryAssets).mockRejectedValueOnce(new Error('连接失败'))
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioLibrary onOpenSession={() => {}} />
      </QueryClientProvider>,
    )
    await screen.getByRole('button', { name: '重试读取' }).click()
    await expect.element(screen.getByText('故事设定.md')).toBeVisible()
  })

  it('shows upload failure with a retry action', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioLibrary onOpenSession={() => {}} />
      </QueryClientProvider>,
    )
    await page.getByLabelText('选择上传资产').upload(new File(['hello'], 'notes.md', { type: 'text/markdown' }))
    await expect.element(screen.getByRole('alert')).toHaveTextContent('网络已断开')
    await expect.element(screen.getByRole('button', { name: '重试上传' })).toBeVisible()
  })
})

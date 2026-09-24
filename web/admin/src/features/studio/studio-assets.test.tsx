import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { StudioAssets } from './studio-assets'

vi.mock('@/lib/api/studio', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/studio')>()),
  listStudioLibraryFolders: vi.fn(async () => []),
}))

describe('StudioAssets', () => {
  it('keeps a new document draft when saving fails', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false, refetchInterval: false } },
    })
    const onCreateTextAsset = vi.fn(async () => {
      throw new Error('服务端暂时不可用')
    })
    const screen = await render(
      <QueryClientProvider client={client}>
        <StudioAssets
          assets={[]}
          onSaveToLibrary={async () => {}}
          onCreateTextAsset={onCreateTextAsset}
        />
      </QueryClientProvider>
    )
    await screen.getByRole('button', { name: '新建文档' }).click()
    await screen.getByRole('textbox', { name: '文档内容' }).fill('# 故事草稿')
    await screen.getByRole('button', { name: '创建文档' }).click()

    await expect
      .element(screen.getByRole('alert'))
      .toHaveTextContent('服务端暂时不可用')
    await expect
      .element(screen.getByRole('textbox', { name: '文档内容' }))
      .toHaveValue('# 故事草稿')
  })
})

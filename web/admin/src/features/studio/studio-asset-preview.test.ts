import { createElement } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import type { StudioAsset } from '@/lib/api/studio'
import { AssetCard, StudioAssets } from './studio-assets'

const createdAt = '2026-09-26T10:00:00Z'

function mediaAsset(kind: 'video' | 'audio'): StudioAsset {
  return {
    id: `${kind}-asset`,
    session_id: 'session-1',
    name: kind === 'video' ? '分镜.mp4' : '旁白.mp3',
    kind,
    origin: 'user',
    current_version: 1,
    saved_to_library: false,
    versions: [
      {
        id: `${kind}-version`,
        version: 1,
        mime_type: kind === 'video' ? 'video/mp4' : 'audio/mpeg',
        size_bytes: 1024,
        content_url: `/api/v1/studio/assets/${kind}-asset/content`,
        created_at: createdAt,
      },
    ],
    created_at: createdAt,
    updated_at: createdAt,
  }
}

function imageAsset(index: number): StudioAsset {
  return {
    id: `image-${index}`,
    session_id: 'session-1',
    name: `图片 ${index}.png`,
    kind: 'image',
    origin: 'user',
    current_version: 1,
    saved_to_library: false,
    versions: [
      {
        id: `image-version-${index}`,
        version: 1,
        mime_type: 'image/png',
        size_bytes: 1024,
        content_url: '/images/favicon.png',
        created_at: createdAt,
      },
    ],
    created_at: createdAt,
    updated_at: createdAt,
  }
}

describe('会话资产预览', () => {
  it('切换网格视图、列表视图和单列视图', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { staleTime: Infinity } },
    })
    client.setQueryData(['studio', 'library', 'categories'], [])
    const screen = await render(
      createElement(
        'div',
        { className: 'h-[600px] w-[600px]' },
        createElement(
          QueryClientProvider,
          { client },
          createElement(StudioAssets, {
            assets: [
              imageAsset(1),
              imageAsset(2),
              imageAsset(3),
              imageAsset(4),
            ],
            onSaveToLibrary: async () => {},
            onUploadAsset: () => {},
            onCreateTextAsset: async () => {},
          })
        )
      )
    )
    const rowCount = () =>
      new Set(
        Array.from(screen.container.querySelectorAll('article')).map((card) =>
          Math.round(card.getBoundingClientRect().top)
        )
      ).size

    const layoutButton = screen.getByRole('button', { name: '网格视图' })
    expect(layoutButton).toBeVisible()
    expect(layoutButton.element().getBoundingClientRect().height).toBe(32)
    expect(
      screen
        .getByRole('button', { name: '上传资产' })
        .element()
        .getBoundingClientRect().left
    ).toBeLessThan(layoutButton.element().getBoundingClientRect().left)
    expect(
      screen
        .getByRole('button', { name: '新建文档' })
        .element()
        .getBoundingClientRect().left
    ).toBeLessThan(layoutButton.element().getBoundingClientRect().left)
    expect(
      getComputedStyle(
        screen.container.querySelector('[data-slot="studio-assets-toolbar"]')!
      ).paddingTop
    ).toBe('4px')
    expect(screen.getByText('共 4 项资产')).toBeVisible()
    expect(rowCount()).toBe(2)
    const cards = screen.container.querySelectorAll('article')
    expect(cards[3].getBoundingClientRect().width).toBeCloseTo(
      cards[0].getBoundingClientRect().width,
      0
    )
    await screen.getByRole('button', { name: '网格视图' }).click()
    await screen.getByRole('menuitemradio', { name: '单列视图' }).click()
    await expect
      .element(screen.getByRole('button', { name: '单列视图' }))
      .toBeVisible()
    expect(rowCount()).toBe(4)
    await screen.getByRole('button', { name: '单列视图' }).click()
    await screen.getByRole('menuitemradio', { name: '列表视图' }).click()
    await expect
      .element(screen.getByRole('button', { name: '列表视图' }))
      .toBeVisible()
    const card = screen.container.querySelector('article')!
    const image = card.querySelector('img')!
    expect(rowCount()).toBe(4)
    expect(image.getBoundingClientRect().width).toBeLessThan(
      card.getBoundingClientRect().width / 2
    )
    await screen.getByRole('button', { name: '列表视图' }).click()
    await screen.getByRole('menuitemradio', { name: '网格视图' }).click()
    expect(rowCount()).toBe(2)
    const container = screen.container.firstElementChild as HTMLElement
    container.style.width = '320px'
    expect(rowCount()).toBe(4)
    container.style.width = '900px'
    expect(rowCount()).toBe(1)
  })

  it('只有一项资产时自适应卡片仍按面板宽度分列', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { staleTime: Infinity } },
    })
    client.setQueryData(['studio', 'library', 'categories'], [])
    const screen = await render(
      createElement(
        'div',
        { className: 'h-[600px] w-[440px]' },
        createElement(
          QueryClientProvider,
          { client },
          createElement(StudioAssets, {
            assets: [imageAsset(1)],
            onSaveToLibrary: async () => {},
          })
        )
      )
    )
    const card = screen.container.querySelector('article')!
    expect(card.getBoundingClientRect().width).toBeLessThan(220)
  })

  it('查看并定位生成资产，用户资产不显示定位操作', async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { staleTime: Infinity } },
    })
    client.setQueryData(['studio', 'library', 'categories'], [])
    const generated = {
      ...imageAsset(1),
      origin: 'agent' as const,
      source_run_id: 'run-1',
    }
    const manual = imageAsset(2)
    let locatedMessageId = ''
    const screen = await render(
      createElement(
        'div',
        { className: 'h-[600px] w-[440px]' },
        createElement(
          QueryClientProvider,
          { client },
          createElement(StudioAssets, {
            assets: [generated, manual],
            messages: [
              {
                id: 'user-1',
                session_id: 'session-1',
                run_id: 'run-1',
                role: 'user',
                content: [{ type: 'text', text: '生成图片' }],
                created_at: createdAt,
              },
              {
                id: 'user-manual',
                session_id: 'session-1',
                role: 'user',
                content: [{ type: 'text', text: '上传图片' }],
                created_at: createdAt,
              },
            ],
            onLocateMessage: (id: string) => {
              locatedMessageId = id
            },
            onSaveToLibrary: async () => {},
          })
        )
      )
    )
    const cards = screen.container.querySelectorAll('article')
    const generatedCard = cards[0]
    const manualCard = cards[1]
    const moreButton = generatedCard.querySelector<HTMLButtonElement>(
      'button[aria-label="更多操作"]'
    )!
    const locateButton = generatedCard.querySelector<HTMLButtonElement>(
      'button[aria-label="定位生成对话"]'
    )!
    expect(getComputedStyle(moreButton).display).toBe('flex')
    expect(getComputedStyle(locateButton.parentElement!).display).toBe('none')
    expect(
      manualCard.querySelector('button[aria-label="定位生成对话"]')
    ).toBeNull()
    const viewButton = screen.getByRole('button', { name: '查看' }).first()
    expect(viewButton.element().getBoundingClientRect().height).toBe(32)
    await screen.getByRole('button', { name: '更多操作' }).hover()
    await expect
      .element(screen.getByRole('tooltip'))
      .toHaveTextContent('更多操作')
    await screen.getByRole('button', { name: '更多操作' }).click()
    await screen.getByRole('menuitem', { name: '定位生成对话' }).click()
    expect(locatedMessageId).toBe('user-1')
    expect(generatedCard.querySelector('a[download]')).not.toBeNull()
    const container = screen.container.firstElementChild as HTMLElement
    container.style.width = '320px'
    expect(getComputedStyle(moreButton).display).toBe('none')
    expect(getComputedStyle(locateButton.parentElement!).display).toBe('flex')
    await screen.getByRole('button', { name: '定位生成对话' }).hover()
    await expect
      .element(screen.getByRole('tooltip'))
      .toHaveTextContent('定位生成对话')
    await viewButton.click()
    await expect
      .element(screen.getByRole('dialog', { name: generated.name }))
      .toBeVisible()
    await screen.getByRole('button', { name: '关闭' }).hover()
    await expect.element(screen.getByRole('tooltip')).toHaveTextContent('关闭')
  })

  it('资产库卡片显示来源定位，不显示存入资产库操作', async () => {
    const asset = {
      ...imageAsset(1),
      origin: 'agent' as const,
      source_run_id: 'run-1',
      saved_to_library: true,
    }
    let located = false
    const screen = await render(
      createElement(
        'div',
        { className: 'w-[440px]' },
        createElement(AssetCard, {
          asset,
          preview: true,
          onLocateSource: () => {
            located = true
          },
        })
      )
    )
    const card = screen.container.querySelector('article')!
    expect(card.querySelector('[aria-label="已存入资产库"]')).toBeNull()
    await screen.getByRole('button', { name: '定位生成对话' }).click()
    expect(located).toBe(true)
  })

  it('窄卡片在更多操作中显示文档编辑', async () => {
    const asset: StudioAsset = {
      ...imageAsset(1),
      id: 'document-asset',
      name: '创作笔记.md',
      kind: 'document',
      versions: [
        {
          ...imageAsset(1).versions[0],
          mime_type: 'text/markdown',
          content_url: '/api/v1/studio/assets/document-asset/content',
        },
      ],
    }
    const client = new QueryClient({
      defaultOptions: { queries: { staleTime: Infinity } },
    })
    client.setQueryData(['studio', 'library', 'categories'], [])
    client.setQueryData(
      ['studio', 'asset', asset.id, asset.versions[0].content_url, 'content'],
      '# 创作笔记'
    )
    const screen = await render(
      createElement(
        'div',
        { className: 'h-[600px] w-[440px]' },
        createElement(
          QueryClientProvider,
          { client },
          createElement(StudioAssets, {
            assets: [asset],
            onSaveToLibrary: async () => {},
            onUpdateTextAsset: async () => {},
          })
        )
      )
    )
    await screen.getByRole('button', { name: '更多操作' }).click()
    await expect
      .element(screen.getByRole('menuitem', { name: '编辑文档' }))
      .toBeVisible()
  })

  it.each(['video', 'audio'] as const)('在卡片中预览 %s 资产', (kind) => {
    const markup = renderToStaticMarkup(
      createElement(
        QueryClientProvider,
        { client: new QueryClient() },
        createElement(StudioAssets, {
          assets: [mediaAsset(kind)],
          onSaveToLibrary: async () => {},
        })
      )
    )
    const preview = new DOMParser()
      .parseFromString(markup, 'text/html')
      .querySelector('.aspect-\\[16\\/10\\]')
    expect(preview).not.toBeNull()
    expect(preview?.querySelector(`${kind}[controls]`)).not.toBeNull()
  })

  it.each(['text/markdown', 'text/plain'] as const)(
    '渲染并滚动 %s 文档预览',
    async (mimeType) => {
      const contentURL = '/api/v1/studio/assets/document-asset/content'
      const asset: StudioAsset = {
        id: 'document-asset',
        session_id: 'session-1',
        name: '创作笔记.md',
        kind: 'document',
        origin: 'user',
        current_version: 1,
        saved_to_library: false,
        versions: [
          {
            id: 'document-version',
            version: 1,
            mime_type: mimeType,
            size_bytes: 22,
            content_url: contentURL,
            created_at: createdAt,
          },
        ],
        created_at: createdAt,
        updated_at: createdAt,
      }
      const client = new QueryClient({
        defaultOptions: { queries: { staleTime: Infinity } },
      })
      client.setQueryData(
        ['studio', 'asset', asset.id, contentURL, 'content'],
        `# 故事设定\n\n**夜雨**\n\n${'分镜描述。\n\n'.repeat(30)}`
      )

      const screen = await render(
        createElement(
          'div',
          { className: 'w-[320px]' },
          createElement(
            QueryClientProvider,
            { client },
            createElement(AssetCard, {
              asset,
              preview: true,
              onSaveToLibrary: () => {},
            })
          )
        )
      )
      const preview = screen.container.querySelector('.aspect-\\[16\\/10\\]')
      expect(preview?.querySelector('h1')?.textContent).toBe('故事设定')
      expect(
        preview?.querySelector('[data-streamdown="strong"]')?.textContent
      ).toBe('夜雨')
      const scroll = Array.from(preview?.querySelectorAll('div') ?? []).find(
        (element) => getComputedStyle(element).overflowY === 'auto'
      ) as HTMLElement | undefined
      expect(scroll).toBeDefined()
      expect(scroll!.scrollHeight).toBeGreaterThan(scroll!.clientHeight)
      scroll!.scrollTop = 50
      expect(scroll!.scrollTop).toBe(50)
    }
  )
})

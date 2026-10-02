import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import { formatDate, formatSize, ProjectTreeBranch, TreeNode } from './studio-library'
import type { StudioAsset, StudioLibraryTreeMode } from '@/lib/api/studio'
import '@/styles/index.css'

function LibraryTree() {
  const [projectId, setProjectId] = useState<string>()
  const [focusId, setFocusId] = useState<string>()
  return <ProjectTreeBranch
    project={{ id: '', name: '未归属项目', asset_count: 0 }}
    mode='asset'
    selectedProjectId={projectId}
    treeFocusId={focusId}
    onFocusTreeItem={setFocusId}
    isFirstProject
    onProject={setProjectId}
    onAsset={() => { throw new Error('空项目不应包含资产') }}
  />
}

describe('资产库文件树', () => {
  it.each<{ kind: StudioAsset['kind']; icon: string }>([
    { kind: 'image', icon: 'image' },
    { kind: 'document', icon: 'file-text' },
    { kind: 'video', icon: 'video' },
    { kind: 'audio', icon: 'audio-lines' },
    { kind: 'data', icon: 'file-braces' },
    { kind: 'file', icon: 'file' },
  ])('$kind 资产在各分组中保持类型图标', async ({ kind, icon }) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const modes: StudioLibraryTreeMode[] = ['asset', 'session', 'category', 'format', 'rating', 'tag']
    for (const mode of modes) {
      const screen = await render(<QueryClientProvider client={client}>
        <TreeNode
          node={{ id: 'asset:entry', label: kind, count: 0, kind: 'asset', asset_id: 'entry', asset_kind: kind }}
          projectId=''
          mode={mode}
          onFocusTreeItem={() => {}}
          onAsset={() => {}}
        />
      </QueryClientProvider>)
      const button = screen.getByRole('button', { name: kind, exact: true }).element()
      expect(button.querySelector(`svg.lucide-${icon}`), mode).not.toBeNull()
      expect(getComputedStyle(button).fontWeight).toBe('400')
      await screen.unmount()
    }
  })

  it('项目名称使用常规字重', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const screen = await render(<QueryClientProvider client={client}><LibraryTree /></QueryClientProvider>)
    expect(getComputedStyle(screen.getByRole('button', { name: '未归属项目', exact: true }).element()).fontWeight).toBe('400')
  })

  it('再次点击项目名称收起内容，并支持连续切换', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const screen = await render(<QueryClientProvider client={client}><LibraryTree /></QueryClientProvider>)
    const project = screen.getByRole('treeitem')
    const name = screen.getByRole('button', { name: '未归属项目', exact: true })
    await expect.element(project).toHaveAttribute('aria-expanded', 'false')
    await name.click()
    await expect.element(project).toHaveAttribute('aria-expanded', 'true')
    const content = project.element().querySelector<HTMLElement>('[data-slot="collapsible-content"]')!
    expect(getComputedStyle(content).animationName).toBe('collapsible-down')
    expect(getComputedStyle(content).animationDuration).toBe('0.15s')
    await name.click()
    await expect.element(project).toHaveAttribute('aria-expanded', 'false')
    expect(getComputedStyle(content).animationName).toBe('collapsible-up')
    expect(content.inert).toBe(true)
    await expect.element(content).not.toBeVisible()
    await name.click()
    await expect.element(project).toHaveAttribute('aria-expanded', 'true')
    await screen.getByRole('button', { name: '收起未归属项目' }).click()
    await expect.element(project).toHaveAttribute('aria-expanded', 'false')
  })
})

describe('资产详情中的文件信息', () => {
  it('缺少或无效日期时保持空值显示', () => {
    expect(formatDate(null)).toBe('—')
    expect(formatDate('invalid')).toBe('—')
  })

  it('按字节数显示文件大小', () => {
    expect(formatSize(512)).toBe('512 B')
    expect(formatSize(1024)).toBe('1.00 KB')
    expect(formatSize(1024 ** 2)).toBe('1.00 MB')
    expect(formatSize(1024 ** 3)).toBe('1.00 GB')
  })
})

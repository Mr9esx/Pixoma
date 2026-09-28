import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import '@/styles/index.css'
import { expect, it } from 'vitest'
import { render } from 'vitest-browser-react'
import type { StudioAsset, StudioSkillSummary } from '@/lib/api/studio'
import { AssetPicker, SkillPicker, WorkflowPicker } from './studio-chat'

const skills: StudioSkillSummary[] = [
  {
    id: 'storyboard',
    name: '分镜助手',
    description: '生成分镜',
    version: '1',
    enabled: true,
    created_at: '',
    updated_at: '',
  },
  {
    id: 'character',
    name: '角色设定',
    description: '设计角色',
    version: '1',
    enabled: true,
    created_at: '',
    updated_at: '',
  },
  {
    id: 'disabled',
    name: '未启用技能',
    description: '',
    version: '1',
    enabled: false,
    created_at: '',
    updated_at: '',
  },
]

it('searches enabled skills and clears the search after selection', async () => {
  function Example() {
    const [open, setOpen] = useState(false)
    const [selected, setSelected] = useState<string[]>([])
    return (
      <SkillPicker
        open={open}
        onOpenChange={setOpen}
        skills={skills}
        value={selected}
        onInsert={(skill) => setSelected((current) => [...current, skill.id])}
      />
    )
  }

  const screen = await render(<Example />)
  await screen.getByRole('button', { name: '选择技能' }).click()
  await screen.getByPlaceholder('搜索技能').fill('分镜')
  await expect.element(screen.getByText('分镜助手')).toBeVisible()
  await expect.element(screen.getByText('角色设定')).not.toBeInTheDocument()
  await screen.getByText('分镜助手').click()

  await screen.getByRole('button', { name: '选择技能，已选 1 项' }).click()
  await expect.element(screen.getByPlaceholder('搜索技能')).toHaveValue('')
  await expect.element(screen.getByText('角色设定')).toBeVisible()
  await expect.element(screen.getByText('未启用技能')).not.toBeInTheDocument()
})

it('filters session assets', async () => {
  const assets: StudioAsset[] = ['故事设定.md', '角色设定.md'].map(
    (name, index) => ({
      id: `asset-${index}`,
      session_id: 'session-1',
      name,
      kind: 'document',
      origin: 'user',
      current_version: 1,
      saved_to_library: false,
      versions: [
        {
          id: `version-${index}`,
          version: 1,
          mime_type: 'text/markdown',
          size_bytes: 1,
          content_url: '',
          created_at: '',
        },
      ],
      created_at: '',
      updated_at: '',
    })
  )
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  function Example() {
    const [assetOpen, setAssetOpen] = useState(false)
    return (
      <QueryClientProvider client={client}>
        <AssetPicker
          open={assetOpen}
          onOpenChange={setAssetOpen}
          assets={assets}
          value={[]}
          onInsert={() => {}}
        />
      </QueryClientProvider>
    )
  }

  const screen = await render(<Example />)
  await screen.getByRole('button', { name: '选择资产' }).click()
  await screen.getByRole('textbox', { name: '搜索资产' }).fill('故事')
  await expect.element(screen.getByText('故事设定.md')).toBeVisible()
  await expect.element(screen.getByText('角色设定.md')).not.toBeInTheDocument()
})

it('opens workflow search', async () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  function Example() {
    const [open, setOpen] = useState(false)
    return (
      <QueryClientProvider client={client}>
        <WorkflowPicker
          open={open}
          onOpenChange={setOpen}
          disabled={false}
          onSelect={() => {}}
        />
      </QueryClientProvider>
    )
  }

  const screen = await render(<Example />)
  await screen.getByRole('button', { name: '选择工作流' }).click()
  await expect.element(screen.getByPlaceholder('搜索工作流')).toBeVisible()
})

import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const read = (path: string) =>
  readFileSync(new URL(path, import.meta.url), 'utf8')

describe('Studio production workspace contract', () => {
  it('uses the official assistant-ui AG-UI runtime and Pixoma primitives', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("from '@assistant-ui/react-ag-ui'")
    expect(source).toContain("from '@assistant-ui/react'")
    expect(source).toContain("from '@ag-ui/client'")
    expect(source).toContain('ComposerPrimitive.Input')
    expect(source).toContain('ThreadPrimitive.Messages')
    expect(source).toContain('@/components/ui/button')
  })

  it('renders an editable React Flow asset road and session assets', () => {
    const source = read('./studio-flow.tsx')
    expect(source).toContain("from '@xyflow/react'")
    expect(source).toContain('onNodesChange')
    expect(source).toContain('Background')
    expect(source).toContain('Controls')

    const workspace = read('./studio-workspace.tsx')
    expect(workspace).toContain('StudioFlow')
    expect(workspace).toContain('StudioAssets')
    expect(workspace).toContain('StudioLibrary')
  })

  it('keeps the Studio session sidebar while reusing platform sidebar styles', () => {
    expect(read('./studio-workspace.tsx')).toContain('StudioSidebar')
    const sidebar = read('./studio-sidebar.tsx')
    expect(sidebar).toContain("from '@/components/ui/sidebar'")
    expect(sidebar).toContain('<SidebarContent')
    expect(sidebar).toContain('最近对话')
    expect(read('../../routes/_app.tsx')).toContain('StudioLayout')
    expect(read('../../config/menu.ts')).toContain("id: 'studio'")
    expect(read('../../routes/_app/studio.tsx')).toContain('StudioWorkspace')
  })

  it('mounts the Studio sidebar primitives inside their provider', () => {
    const layout = read('../../routes/_app.tsx')
    const studioLayout = layout.slice(layout.indexOf('function StudioLayout'))
    expect(studioLayout).toContain('<SidebarProvider')
  })

  it('composes Studio context with the platform sidebar shell', () => {
    const sidebar = read('./studio-sidebar.tsx')
    expect(sidebar).toContain("from '@/components/layout/app-title'")
    expect(sidebar).toContain("from '@/components/layout/nav-user'")
    expect(sidebar).toContain("<Sidebar collapsible='none' className='bg-muted/30 p-2'>")
    expect(sidebar).toContain('<AppTitle showToggle={false} />')
    expect(sidebar).toContain('<NavUser />')
    expect(sidebar).toContain("to='/'")
    expect(sidebar).not.toContain('<aside')
    expect(sidebar).not.toContain('<Avatar')
    expect(sidebar).toContain("isActive={view === 'chat'}")
    expect(sidebar).toContain("className='min-h-0 flex-1 px-2 py-1'")
    expect(sidebar).toContain('<SidebarGroupLabel>最近对话</SidebarGroupLabel>')
    expect(sidebar).toContain("<SidebarMenu className='pb-2'>")
    expect(sidebar).toContain("from '@/components/ui/button'")
    expect(sidebar).toContain('MessageSquarePlus')
    expect(sidebar).toContain("className='mt-2 w-full'")
    expect(sidebar).toContain('onClick={onNewSession}')
    expect(sidebar.indexOf('</ScrollArea>')).toBeLessThan(
      sidebar.indexOf("className='mt-2 w-full'")
    )
    expect(sidebar).not.toContain('SidebarMenuSub')
    expect(sidebar).not.toContain('SidebarGroupAction')
    expect(sidebar).not.toContain("className='inset-e-0 top-1.5'")
    expect(sidebar).not.toContain("className='h-10 w-full")
    expect(sidebar).not.toContain('min-h-11')
    expect(sidebar).not.toContain('line-clamp-2')
  })

  it('groups chat and canvas in one padded rounded workbench without structural borders', () => {
    const source = read('./studio-workspace.tsx')
    expect(source).toContain("data-slot='studio-workbench'")
    expect(source).toContain('rounded-2xl')
    expect(source).not.toContain('max-w-2xl min-w-80 shrink-0 border-l')
  })

  it('keeps the composer as the chat boundary without a full-width chrome strip', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("className='shrink-0 px-4 pt-2 pb-5'")
    expect(source).not.toContain("className='shrink-0 bg-background")
  })

  it('provides a connected Skill configuration view', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain('listStudioSkills')
    expect(source).toContain('createStudioSkill')

    expect(source).toContain('updateStudioSkill')
    expect(source).toContain('SkillDialog')
    expect(source).not.toContain('<EmptySetting icon={Sparkles}')
  })

  it('provides a connected MCP connector configuration view', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain('listStudioConnectors')
    expect(source).toContain('createStudioConnector')
    expect(source).toContain('updateStudioConnector')
    expect(source).toContain('ConnectorDialog')
    expect(source).not.toContain("title='MCP 连接器'")
  })

  it('lists existing workflows and lets an admin set Agent availability', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain('listStudioAgentWorkflows')
    expect(source).toContain('updateStudioAgentWorkflow')
    expect(source).toContain('WorkflowSettings')
    expect(source).not.toContain("title='Agent 可用工作流'")
  })

  it('lets the composer forward explicitly selected Skills', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain('SkillPicker')
    expect(source).toContain('selectedSkillIds')
    expect(source).toContain('selectedSkillIds: props.selectedSkillIds')
  })

  it('lets the composer use both Session assets and reusable library assets', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain('listStudioLibraryAssets')
    expect(source).toContain('当前 Session')
    expect(source).toContain('资产库')
    expect(source).toContain('new Map<string, StudioAsset>')
  })
})

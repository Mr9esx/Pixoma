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
})

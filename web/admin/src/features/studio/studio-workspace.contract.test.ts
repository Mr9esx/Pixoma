import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

const read = (path: string) =>
  readFileSync(new URL(path, import.meta.url), 'utf8')

describe('Studio production workspace contract', () => {
  it('uses the AG-UI runtime bridge with AI Elements as the chat presentation layer', () => {
    const source = read('./studio-chat.tsx')
    const transport = read('../../lib/agui-websocket-agent.ts')
    expect(source).toContain("from '@assistant-ui/react-ag-ui'")
    expect(source).toContain("from '@assistant-ui/react'")
    expect(transport).toContain("from '@ag-ui/client'")
    expect(source).toContain("from '@/components/ai-elements/conversation'")
    expect(source).toContain("from '@/components/ai-elements/message'")
    expect(source).toContain("from '@/components/ai-elements/prompt-input'")
    expect(source).toContain('ConversationContent')
    expect(source).toContain('MessageResponse')
    expect(source).toContain('PromptInputTextarea')
    expect(source).not.toContain('ComposerPrimitive.Input')
    expect(source).not.toContain('ThreadPrimitive.Messages')
    expect(source).toContain("from '@/components/ui/dropdown-menu'")
  })

  it('renders an editable React Flow asset road and session assets', () => {
    const source = read('./studio-flow.tsx')
    expect(source).toContain("from '@xyflow/react'")
    expect(source).toContain('onNodesChange')
    expect(source).toContain('onNodesDelete')
    expect(source).toContain('onEdgesDelete')
    expect(source).toContain('onConnect')
    expect(source).toContain('CreateFlowNodeDialog')
    expect(source).toContain('Background')
    expect(source).toContain('Controls')

    const workspace = read('./studio-workspace.tsx')
    expect(workspace).toContain('StudioFlow')
    expect(workspace).toContain('StudioAssets')
    expect(workspace).toContain('StudioLibrary')
    expect(workspace).toContain('uploadStudioAsset')

    const assets = read('./studio-assets.tsx')
    expect(assets).toContain('上传资产')
    expect(assets).toContain("type='file'")
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

  it('reconciles active runs when the page becomes visible or online again', () => {
    const workspace = read('./studio-workspace.tsx')
    const sidebar = read('./studio-sidebar.tsx')
    expect(workspace).toContain("refetchOnMount: 'always'")
    expect(workspace).toContain('refetchOnWindowFocus: true')
    expect(workspace).toContain("addEventListener('visibilitychange'")
    expect(workspace).toContain("addEventListener('online'")
    expect(workspace).toContain('onRuntimeStateChange')
    expect(sidebar).toContain('StatusDot')
    expect(sidebar).toContain('waiting_approval')
    expect(sidebar).toContain('succeeded')
  })

  it('keeps the chat mounted while refreshing and replays a new runtime from event zero', () => {
    const workspace = read('./studio-workspace.tsx')
    const chat = read('./studio-chat.tsx')
    expect(workspace).not.toContain('(detail.isFetching &&')
    expect(chat).not.toContain(
      'afterSequence: props.runProgress?.last_sequence'
    )
    expect(chat).toContain('afterSequence: 0')
  })

  it('sends stop to the Studio backend before ending the local run', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain('cancelStudioRun(runID)')
    expect(source).toContain('agent.activeStudioRunId()')
  })

  it('blocks a second send while the server still owns an active run', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("props.latestRun?.status === 'waiting_approval'")
    expect(source).toContain(
      'if (!modelReady || runActive || !text.trim()) return'
    )
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
    expect(sidebar).toContain("<Sidebar collapsible='none' className='p-2'>")
    expect(sidebar).toContain('<AppTitle showToggle={false} />')
    expect(sidebar).toContain('<NavUser />')
    expect(sidebar).toContain("to='/'")
    expect(sidebar).not.toContain('<aside')
    expect(sidebar).not.toContain('<Avatar')
    expect(sidebar).toContain("<Sidebar collapsible='none' className='p-2'>")
    expect(sidebar).toContain("isActive={view === 'chat'}")
    expect(sidebar).toContain("className='min-h-0 flex-1 ps-2 pe-0 py-1'")
    expect(sidebar).toContain('<SidebarGroupLabel>最近对话</SidebarGroupLabel>')
    expect(sidebar).toContain(
      "<SidebarMenu className='w-full min-w-0 self-stretch pb-2'>"
    )
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

  it('floats the composer over the conversation so messages pass behind it', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("data-slot='studio-composer'")
    expect(source).toContain(
      'pointer-events-none absolute inset-x-0 bottom-0 z-20'
    )
    expect(source).toContain('bg-gradient-to-t from-card to-transparent')
    expect(source).toContain('pb-44')
    expect(source).toContain("aria-label='跳转至最新消息'")
    expect(source).toContain("className='bottom-48'")
    expect(source).not.toContain("className='bottom-44'")
    expect(source).not.toContain("className='shrink-0 px-4 pt-2 pb-5'")
  })

  it('keeps the composer focus ring identical to the other admin inputs', () => {
    const group = read('../../components/ui/input-group.tsx')
    expect(group).toContain(
      'has-[[data-slot=input-group-control]:focus-visible]:border-ring'
    )
    expect(group).toContain(
      'has-[[data-slot=input-group-control]:focus-visible]:ring-1'
    )
    expect(group).not.toContain('ring-[3px]')
    expect(read('../../components/ui/input.tsx')).toContain(
      'focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring/50'
    )
  })

  it('keeps every composer control on the design system radius scale', () => {
    const group = read('../../components/ui/input-group.tsx')
    expect(group).toContain('rounded-md')
    expect(group).not.toContain('rounded-lg')
    expect(group).not.toContain('rounded-xl')
    const source = read('./studio-chat.tsx')
    expect(source).not.toContain('rounded-lg')
  })

  it('switches the model from a dropdown menu instead of a modal', () => {
    const source = read('./studio-chat.tsx')
    expect(source).not.toContain('ModelSelector')
    expect(source).not.toContain('model-selector')
    const pickup = source.slice(
      source.indexOf('function ModelPicker'),
      source.indexOf('const permissionLabels')
    )
    expect(pickup).toContain('<DropdownMenu>')
    expect(pickup).toContain('<DropdownMenuRadioGroup value={value}')
  })

  it('keeps Skills and assets as icon buttons that explain themselves on hover', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("size='icon-sm'")
    expect(source).toContain("tooltip={value.length === 0 ? 'Skills'")
    expect(source).toContain("tooltip={value.length === 0 ? '资产'")
  })

  it('keeps the copy inside the composer input at normal weight', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("className='max-w-52 font-normal'")
    expect(source).toContain("className='font-normal'")
  })

  it('groups the model switcher with the send button on the right of the footer', () => {
    const source = read('./studio-chat.tsx')
    const leftTools = source.slice(
      source.indexOf('<PromptInputTools>'),
      source.lastIndexOf('<PromptInputTools>')
    )
    expect(leftTools).toContain('<SkillPicker')
    expect(leftTools).toContain('<AssetPicker')
    expect(leftTools).toContain('<PermissionPicker')
    expect(leftTools).not.toContain('<ModelPicker')
    expect(leftTools.indexOf('<AssetPicker')).toBeLessThan(
      leftTools.indexOf('<PermissionPicker')
    )
    const rightTools = source.slice(source.lastIndexOf('<PromptInputTools>'))
    expect(rightTools).toContain('<ModelPicker')
    expect(rightTools).toContain('<PromptInputSubmit')
    expect(rightTools.indexOf('<ModelPicker')).toBeLessThan(
      rightTools.indexOf('<PromptInputSubmit')
    )
  })

  it('replaces chat and composer with session trajectory without opening a drawer', () => {
    const source = read('./studio-workspace.tsx')
    expect(source).toContain('traceOpen && sessionId')
    expect(source).toContain(
      '<StudioTrace key={sessionId} sessionId={sessionId} />'
    )
    expect(source).toContain('rightOpen && !traceOpen')
    expect(source).not.toContain('<Sheet open={traceOpen}')
  })

  it('provides a connected Skill configuration view', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain(
      "className='hidden w-60 shrink-0 border-e bg-muted/20 p-3 md:block'"
    )
    expect(source).toContain("aria-label='AI 设置分类'")
    expect(source).not.toContain("from '@/components/ui/tabs'")
    expect(source).toContain('listStudioSkills')
    expect(source).toContain('createStudioSkill')

    expect(source).toContain('updateStudioSkill')
    expect(source).toContain('SkillDialog')
    expect(source).not.toContain('<EmptySetting icon={Sparkles}')
  })

  it('lets administrators validate saved model connections without exposing their keys', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain('testStudioModelConnection')
    expect(source).toContain('测试连接')
    expect(source).toContain('连接成功')
    expect(source).toContain("import { toast } from 'sonner'")
    expect(source).toContain('toast.error')
  })

  it('lets administrators test the unsaved add-model form from the dialog footer', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain('testStudioModelConfig')
    expect(source).toContain('测试配置')
    expect(source).toContain('正在测试…')
    expect(source).toContain('DialogFooter')
    expect(source).toContain('reportValidity')
  })

  it('provides an edit entry for saved models and reuses the model dialog', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain('updateStudioModel')
    expect(source).toContain('编辑')
    expect(source).toContain('editingModel')
    expect(source).toContain('ModelDialog')
  })

  it('provides a connected MCP connector configuration view', () => {
    const source = read('./studio-settings.tsx')
    expect(source).toContain('listStudioConnectors')
    expect(source).toContain('createStudioConnector')
    expect(source).toContain('updateStudioConnector')
    expect(source).toContain('ConnectorDialog')
    expect(source).toContain('probeStudioConnector')
    expect(source).toContain('发现工具')
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

  it('renders assistant text, reasoning and tool calls with AI Elements', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain('MessageResponse')
    expect(source).toContain('ReasoningContent')
    expect(source).toContain('ToolHeader')
  })

  it('renders AG-UI approval interrupts with an actionable AI Elements confirmation', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain('useAgUiInterrupts')
    expect(source).toContain('useAgUiSubmitInterruptResponses')
    expect(source).toContain("from '@/components/ai-elements/confirmation'")
    expect(source).toContain('ConfirmationAction')
    expect(source).toContain("status: approved ? 'resolved' : 'cancelled'")
    expect(source).toContain('reason: interrupt.reason')
  })

  it('titles the approval alert with the request type and puts the pending action beside the buttons', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain(
      "import { AlertDescription, AlertTitle } from '@/components/ui/alert'"
    )
    expect(source).toContain(
      "reason === 'tool_approval' ? '权限审批' : '需要处理'"
    )
    expect(source).toContain(
      '<AlertTitle>{approvalTitle(action.reason)}</AlertTitle>'
    )
    expect(source).toContain("<div className='flex w-full items-center gap-3'>")
    expect(source).toContain("<AlertDescription className='flex-1'>")
    expect(source).toContain(
      "<ConfirmationActions className='shrink-0 self-center'>"
    )
    expect(source).toContain("{action.message ?? '需要批准后继续执行'}")
  })

  it('covers the composer input with the approval actions while the agent waits', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain("aria-label='操作区'")
    expect(source).toContain("variant='warn'")
    expect(source).toContain(
      'absolute inset-0 z-10 flex flex-col gap-2 bg-card'
    )
    expect(source).toContain("className='flex-1 justify-center'")
    expect(source).toContain(
      "<div className='pointer-events-auto relative z-10'>"
    )
    expect(source).toContain(
      "<div className='mx-auto flex w-full max-w-3xl flex-col px-5 pb-5'>"
    )
    expect(source).not.toContain('StudioActionDock')
    expect(source).not.toContain('<StudioApprovalPrompt')
    expect(source.indexOf('<StudioActionArea />')).toBeLessThan(
      source.indexOf('<PromptInput')
    )
  })

  it('enables Streamdown animation for streaming answers and reasoning', () => {
    const message = read('../../components/ai-elements/message.tsx')
    const reasoning = read('../../components/ai-elements/reasoning.tsx')
    expect(message).toContain(
      'animated={animated ?? (isAnimating ? true : undefined)}'
    )
    expect(reasoning).toContain('useReasoning()')
    expect(reasoning).toContain('animated={isStreaming ? true : undefined}')
  })

  it('renders AG-UI run errors in the conversation instead of dropping them', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain('onError: (error) => setRunError(error.message)')
    expect(source).toContain('runError')
    expect(source).toContain("role='alert'")
  })

  it('lets the composer use both Session assets and reusable library assets', () => {
    const source = read('./studio-chat.tsx')
    expect(source).toContain('listStudioLibraryAssets')
    expect(source).toContain('当前 Session')
    expect(source).toContain('资产库')
    expect(source).toContain('new Map<string, StudioAsset>')
  })

  it('lets a Session asset choose its library folder before saving', () => {
    const assets = read('./studio-assets.tsx')
    const workspace = read('./studio-workspace.tsx')

    expect(assets).toContain('listStudioLibraryFolders')
    expect(assets).toContain('SaveAssetToLibraryDialog')
    expect(assets).toContain('选择资产库文件夹')
    expect(workspace).toContain('saveAsset.mutateAsync')
  })

  it('edits Session documents by appending an asset version', () => {
    const assets = read('./studio-assets.tsx')
    const workspace = read('./studio-workspace.tsx')

    expect(assets).toContain('TextAssetEditDialog')
    expect(assets).toContain('编辑文档')
    expect(workspace).toContain('updateStudioTextAsset')
    expect(workspace).toContain('updateTextAsset.mutate')
  })

  it('derives the initial Studio session from query data without an effect state copy', () => {
    const source = read('./studio-workspace.tsx')

    expect(source).toContain(
      'const sessionId = activeSessionId ?? sessions.data?.[0]?.id'
    )
    expect(source).not.toContain('setActiveSessionId(first.id)')
    expect(source).toContain(
      '[sessions.data, sessions.isLoading, creatingSession, createSessionMutate]'
    )
  })
})

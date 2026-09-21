import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, Library, MessageSquarePlus, PanelRightClose, PanelRightOpen, Settings2 } from 'lucide-react'
import {
  createStudioSession,
  createStudioTextAsset,
  getStudioSession,
  listStudioModels,
  listStudioSessions,
  saveStudioAssetToLibrary,
  updateStudioFlowNodes,
  type StudioPermissionMode,
  type StudioSession,
} from '@/lib/api/studio'
import { StudioAssets } from './studio-assets'
import { StudioChat } from './studio-chat'
import { StudioFlow } from './studio-flow'
import { StudioLibrary } from './studio-library'
import { StudioSettings } from './studio-settings'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

type StudioView = 'chat' | 'library' | 'settings'

export function StudioWorkspace() {
  const queryClient = useQueryClient()
  const [view, setView] = useState<StudioView>('chat')
  const [activeSessionId, setActiveSessionId] = useState<string>()
  const [rightOpen, setRightOpen] = useState(true)
  const [modelConfigId, setModelConfigId] = useState<string>()
  const [permissionMode, setPermissionMode] = useState<StudioPermissionMode>('request_approval')

  const sessions = useQuery({
    queryKey: ['studio', 'sessions'],
    queryFn: () => listStudioSessions({ limit: 80 }),
  })
  const models = useQuery({ queryKey: ['studio', 'models'], queryFn: listStudioModels })
  const createSession = useMutation({
    mutationFn: createStudioSession,
    onSuccess: (session) => {
      setActiveSessionId(session.id)
      setPermissionMode(session.permission_mode)
      setView('chat')
      void queryClient.invalidateQueries({ queryKey: ['studio', 'sessions'] })
    },
  })

  useEffect(() => {
    if (activeSessionId || sessions.isLoading) return
    const first = sessions.data?.[0]
    if (first) {
      setActiveSessionId(first.id)
      setPermissionMode(first.permission_mode)
    } else if (sessions.data && !createSession.isPending) {
      createSession.mutate()
    }
  }, [activeSessionId, sessions.data, sessions.isLoading, createSession])

  const detail = useQuery({
    queryKey: ['studio', 'session', activeSessionId],
    queryFn: () => getStudioSession(activeSessionId!),
    enabled: Boolean(activeSessionId) && view === 'chat',
    refetchInterval: 1500,
  })

  const saveAsset = useMutation({
    mutationFn: (assetId: string) => saveStudioAssetToLibrary(assetId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['studio'] })
    },
  })
  const saveFlowPositions = useMutation({
    mutationFn: (nodes: Array<{ id: string; position: { x: number; y: number }; sort_order: number }>) =>
      updateStudioFlowNodes(activeSessionId!, nodes),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['studio', 'session', activeSessionId] })
    },
  })
  const createTextAsset = useMutation({
    mutationFn: (input: { name: string; content: string }) =>
      createStudioTextAsset({ sessionId: activeSessionId!, ...input }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['studio', 'session', activeSessionId] })
    },
  })

  return (
    <div className='flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl bg-background'>
      <StudioContextBar
        sessions={sessions.data ?? []}
        activeSessionId={activeSessionId}
        view={view}
        creating={createSession.isPending}
        onNewSession={() => createSession.mutate()}
        onSelectSession={(id) => {
          setActiveSessionId(id)
          setView('chat')
        }}
        onViewChange={setView}
      />
      {view === 'library' ? <StudioLibrary /> : null}
      {view === 'settings' ? <StudioSettings /> : null}
      {view === 'chat' ? (
        <>
          <main id='main-content' className='flex min-w-0 flex-1 flex-col'>
            <header className='flex min-h-16 items-center justify-between gap-3 px-5'>
              <div className='min-w-0'>
                <h1 className='truncate text-sm font-semibold'>{detail.data?.session.title ?? '新对话'}</h1>
                <p className='text-xs text-muted-foreground'>自动保存 · 后台运行</p>
              </div>
              <Button variant='ghost' size='icon' onClick={() => setRightOpen((open) => !open)} aria-label={rightOpen ? '收起右侧面板' : '展开右侧面板'}>
                {rightOpen ? <PanelRightClose /> : <PanelRightOpen />}
              </Button>
            </header>
            {!activeSessionId || detail.isLoading ? (
              <ChatSkeleton />
            ) : detail.isError || !detail.data ? (
              <div className='flex flex-1 items-center justify-center text-sm text-muted-foreground'>对话读取失败，请稍后重试。</div>
            ) : (
              <StudioChat
                key={activeSessionId}
                sessionId={activeSessionId}
                messages={detail.data.messages}
                models={models.data ?? []}
                modelConfigId={modelConfigId ?? detail.data.session.model_config_id}
                permissionMode={permissionMode}
                onModelChange={setModelConfigId}
                onPermissionChange={setPermissionMode}
              />
            )}
          </main>
          {rightOpen ? (
            <aside className='hidden min-w-80 w-[42%] max-w-2xl shrink-0 border-l xl:flex xl:flex-col'>
              <Tabs defaultValue='flow' className='min-h-0 flex-1 gap-0'>
                <div className='flex h-16 items-center border-b px-4'><TabsList><TabsTrigger value='flow'>资产路线</TabsTrigger><TabsTrigger value='assets'>Session 资产 <span className='text-xs text-muted-foreground'>{detail.data?.assets.length ?? 0}</span></TabsTrigger></TabsList></div>
                <TabsContent value='flow' className='m-0 min-h-0'><StudioFlow nodes={detail.data?.flow.nodes ?? []} edges={detail.data?.flow.edges ?? []} onPositionsChange={(nodes) => saveFlowPositions.mutate(nodes)} /></TabsContent>
                <TabsContent value='assets' className='m-0 min-h-0'><StudioAssets assets={detail.data?.assets ?? []} onSaveToLibrary={(id) => saveAsset.mutate(id)} onCreateTextAsset={(input) => createTextAsset.mutate(input)} /></TabsContent>
              </Tabs>
            </aside>
          ) : null}
        </>
      ) : null}
    </div>
  )
}

function StudioContextBar({
  sessions,
  activeSessionId,
  view,
  creating,
  onNewSession,
  onSelectSession,
  onViewChange,
}: {
  sessions: StudioSession[]
  activeSessionId?: string
  view: StudioView
  creating: boolean
  onNewSession: () => void
  onSelectSession: (id: string) => void
  onViewChange: (view: StudioView) => void
}) {
  const activeSession = sessions.find((session) => session.id === activeSessionId)
  return (
    <header className='flex min-h-14 shrink-0 items-center justify-between gap-3 px-5 py-2'>
      <div className='flex min-w-0 items-center gap-1'>
        <Button size='sm' onClick={onNewSession} disabled={creating}>
          <MessageSquarePlus />
          {creating ? '正在创建…' : '新对话'}
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant='ghost' size='sm' className='max-w-52'>
              <span className='truncate'>{activeSession?.title ?? '会话历史'}</span>
              <ChevronDown className='size-3.5' />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align='start' className='w-72'>
            <DropdownMenuLabel>最近对话</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {sessions.length === 0 ? <DropdownMenuItem disabled>还没有对话记录</DropdownMenuItem> : null}
            {sessions.map((session) => (
              <DropdownMenuItem key={session.id} onSelect={() => onSelectSession(session.id)}>
                <span className='truncate'>{session.title}</span>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className='flex shrink-0 items-center gap-1'>
        <Button variant={view === 'library' ? 'secondary' : 'ghost'} size='sm' onClick={() => onViewChange('library')}>
          <Library />
          <span className='hidden sm:inline'>资产库</span>
        </Button>
        <Button variant={view === 'settings' ? 'secondary' : 'ghost'} size='sm' onClick={() => onViewChange('settings')}>
          <Settings2 />
          <span className='hidden sm:inline'>AI 设置</span>
        </Button>
      </div>
    </header>
  )
}

function ChatSkeleton() {
  return <div className='mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 p-8'><Skeleton className='mt-10 h-20 w-3/4 self-end rounded-2xl' /><Skeleton className='h-28 w-4/5 rounded-2xl' /><div className='flex-1' /><Skeleton className='h-32 w-full rounded-2xl' /></div>
}

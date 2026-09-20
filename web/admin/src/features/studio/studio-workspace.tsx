import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Menu, PanelRightClose, PanelRightOpen } from 'lucide-react'
import {
  createStudioSession,
  getStudioSession,
  listStudioModels,
  listStudioSessions,
  saveStudioAssetToLibrary,
  type StudioPermissionMode,
} from '@/lib/api/studio'
import { StudioAssets } from './studio-assets'
import { StudioChat } from './studio-chat'
import { StudioFlow } from './studio-flow'
import { StudioLibrary } from './studio-library'
import { StudioSettings } from './studio-settings'
import { StudioSidebar, type StudioView } from './studio-sidebar'
import { Button } from '@/components/ui/button'
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

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

  const sidebarProps = useMemo(
    () => ({
      sessions: sessions.data ?? [],
      activeSessionId,
      view,
      creating: createSession.isPending,
      onNewSession: () => createSession.mutate(),
      onSelectSession: (id: string) => {
        setActiveSessionId(id)
        setView('chat' as const)
      },
      onViewChange: setView,
    }),
    [sessions.data, activeSessionId, view, createSession]
  )

  return (
    <div className='flex h-svh min-h-0 w-full overflow-hidden bg-background'>
      <div className='hidden lg:flex'><StudioSidebar {...sidebarProps} /></div>
      <div className='fixed left-3 top-3 z-30 lg:hidden'>
        <Sheet>
          <SheetTrigger asChild><Button variant='outline' size='icon' aria-label='打开 Studio 菜单'><Menu /></Button></SheetTrigger>
          <SheetContent side='left' className='w-72 p-0'><SheetTitle className='sr-only'>Studio 菜单</SheetTitle><StudioSidebar {...sidebarProps} /></SheetContent>
        </Sheet>
      </div>

      {view === 'library' ? <StudioLibrary /> : null}
      {view === 'settings' ? <StudioSettings /> : null}
      {view === 'chat' ? (
        <>
          <main id='main-content' className='flex min-w-0 flex-1 flex-col'>
            <header className='flex min-h-16 items-center justify-between gap-3 border-b px-5 pl-16 lg:pl-5'>
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
                <TabsContent value='flow' className='m-0 min-h-0'><StudioFlow nodes={detail.data?.flow.nodes ?? []} edges={detail.data?.flow.edges ?? []} /></TabsContent>
                <TabsContent value='assets' className='m-0 min-h-0'><StudioAssets assets={detail.data?.assets ?? []} onSaveToLibrary={(id) => saveAsset.mutate(id)} /></TabsContent>
              </Tabs>
            </aside>
          ) : null}
        </>
      ) : null}
    </div>
  )
}

function ChatSkeleton() {
  return <div className='mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 p-8'><Skeleton className='mt-10 h-20 w-3/4 self-end rounded-2xl' /><Skeleton className='h-28 w-4/5 rounded-2xl' /><div className='flex-1' /><Skeleton className='h-32 w-full rounded-2xl' /></div>
}

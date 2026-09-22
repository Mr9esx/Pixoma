import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ListTree, Menu, PanelRightClose, PanelRightOpen } from 'lucide-react'
import {
  createStudioSession,
  createStudioTextAsset,
  createStudioFlowEdge,
  createStudioFlowNode,
  deleteStudioFlowEdge,
  deleteStudioFlowNode,
  getStudioSession,
  listStudioModels,
  listStudioSkills,
  listStudioSessions,
  saveStudioAssetToLibrary,
  importStudioLibraryAsset,
  uploadStudioAsset,
  updateStudioTextAsset,
  updateStudioFlowNodes,
  type StudioPermissionMode,
} from '@/lib/api/studio'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { StudioAssets } from './studio-assets'
import { StudioChat } from './studio-chat'
import { StudioFlow } from './studio-flow'
import { StudioLibrary } from './studio-library'
import { StudioSettings } from './studio-settings'
import { StudioSidebar, type StudioView } from './studio-sidebar'
import { StudioTrace } from './studio-trace'

type SelectedAsset = { assetId: string; assetVersionId: string }

export function StudioWorkspace() {
  const queryClient = useQueryClient()
  const [view, setView] = useState<StudioView>('chat')
  const [activeSessionId, setActiveSessionId] = useState<string>()
  const [rightOpen, setRightOpen] = useState(true)
  const [traceOpen, setTraceOpen] = useState(false)
  const [modelConfigId, setModelConfigId] = useState<string>()
  const [selectedSkillIds, setSelectedSkillIds] = useState<string[]>([])
	const [selectedAssets, setSelectedAssets] = useState<SelectedAsset[]>([])
  const [permissionMode, setPermissionMode] =
    useState<StudioPermissionMode>('request_approval')

  const sessions = useQuery({
    queryKey: ['studio', 'sessions'],
    queryFn: () => listStudioSessions({ limit: 80 }),
  })
  const models = useQuery({
    queryKey: ['studio', 'models'],
    queryFn: listStudioModels,
  })
  const skills = useQuery({
    queryKey: ['studio', 'skills'],
    queryFn: listStudioSkills,
  })
  const { mutate: createSessionMutate, isPending: creatingSession } =
    useMutation({
    mutationFn: createStudioSession,
    onSuccess: (session) => {
      setActiveSessionId(session.id)
      setPermissionMode(session.permission_mode)
      setView('chat')
      void queryClient.invalidateQueries({ queryKey: ['studio', 'sessions'] })
    },
  })
  const sessionId = activeSessionId ?? sessions.data?.[0]?.id
  const sessionPermissionMode = activeSessionId
    ? permissionMode
    : sessions.data?.[0]?.permission_mode ?? permissionMode

  useEffect(() => {
    if (
      sessions.isLoading ||
      !sessions.data ||
      sessions.data.length > 0 ||
      creatingSession
    ) {
      return
    }
    createSessionMutate()
  }, [sessions.data, sessions.isLoading, creatingSession, createSessionMutate])

  const detail = useQuery({
    queryKey: ['studio', 'session', sessionId],
    queryFn: () => getStudioSession(sessionId!),
    enabled: Boolean(sessionId) && view === 'chat',
  })

  const saveAsset = useMutation({
    mutationFn: ({ assetId, folderId }: { assetId: string; folderId?: string }) =>
      saveStudioAssetToLibrary(assetId, folderId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['studio'] })
    },
  })
  const saveFlowPositions = useMutation({
    mutationFn: (
      nodes: Array<{
        id: string
        position: { x: number; y: number }
        sort_order: number
      }>
    ) => updateStudioFlowNodes(sessionId!, nodes),
    onSettled: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })
  const createFlowNode = useMutation({
    mutationFn: (input: {
      type: 'stage' | 'plan' | 'operation'
      title: string
      body?: string
      position: { x: number; y: number }
    }) => createStudioFlowNode(sessionId!, input),
    onSettled: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })
  const deleteFlowNode = useMutation({
    mutationFn: (nodeId: string) => deleteStudioFlowNode(sessionId!, nodeId),
    onSettled: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })
  const createFlowEdge = useMutation({
    mutationFn: (input: { source: string; target: string; label?: string }) =>
      createStudioFlowEdge(sessionId!, input),
    onSettled: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })
  const deleteFlowEdge = useMutation({
    mutationFn: (edgeId: string) => deleteStudioFlowEdge(sessionId!, edgeId),
    onSettled: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })
  const createTextAsset = useMutation({
    mutationFn: (input: { name: string; content: string }) =>
      createStudioTextAsset({ sessionId: sessionId!, ...input }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })
  const updateTextAsset = useMutation({
    mutationFn: (input: { assetId: string; content: string }) =>
      updateStudioTextAsset(input.assetId, input.content),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })
	const importLibraryAsset = useMutation({
		mutationFn: ({ assetId, assetVersionId }: SelectedAsset) => {
			if (!sessionId) throw new Error('请先创建或选择一个对话')
			return importStudioLibraryAsset(sessionId, assetId, assetVersionId)
		},
		onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['studio', 'session', sessionId] }),
	})
  const uploadAsset = useMutation({
    mutationFn: (file: File) => uploadStudioAsset(file, sessionId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })

  const sidebarProps = {
    sessions: sessions.data ?? [],
    activeSessionId: sessionId,
    view,
    creating: creatingSession,
    onNewSession: () => createSessionMutate(),
    onSelectSession: (id: string) => {
      setActiveSessionId(id)
      setPermissionMode(
        sessions.data?.find((session) => session.id === id)?.permission_mode ??
          'request_approval'
      )
      setView('chat' as const)
    },
    onViewChange: setView,
  }

  return (
    <div className='flex h-svh min-h-0 w-full overflow-hidden bg-muted/30'>
      <div className='hidden lg:flex'>
        <StudioSidebar {...sidebarProps} />
      </div>
      <div className='fixed top-3 left-3 z-30 lg:hidden'>
        <Sheet>
          <SheetTrigger asChild>
            <Button variant='outline' size='icon' aria-label='打开 Studio 菜单'>
              <Menu />
            </Button>
          </SheetTrigger>
          <SheetContent side='left' className='w-64 bg-muted/30 p-0'>
            <SheetTitle className='sr-only'>Studio 菜单</SheetTitle>
            <StudioSidebar {...sidebarProps} />
          </SheetContent>
        </Sheet>
      </div>

      {view === 'library' ? <StudioLibrary /> : null}
      {view === 'settings' ? <StudioSettings /> : null}
      {view === 'chat' ? (
        <div className='min-h-0 min-w-0 flex-1 bg-muted/30 p-3 sm:p-4'>
          <section
            data-slot='studio-workbench'
            className='flex h-full min-h-0 overflow-hidden rounded-2xl border bg-card'
          >
            <main id='main-content' className='flex min-w-0 flex-1 flex-col'>
              <header className='flex min-h-16 items-center justify-between gap-3 px-5 pl-16 lg:pl-5'>
                <div className='min-w-0'>
                  <h1 className='truncate text-sm font-semibold'>
                    {detail.data?.session.title ?? '新对话'}
                  </h1>
                  <p className='text-xs text-muted-foreground'>
                    自动保存 · 后台运行
                  </p>
                </div>
                <div className='flex items-center gap-1'>
                  <Button variant='ghost' size='sm' onClick={() => setTraceOpen(true)}><ListTree />Trace</Button>
                  <Button variant='ghost' size='icon' onClick={() => setRightOpen((open) => !open)} aria-label={rightOpen ? '收起右侧面板' : '展开右侧面板'}>{rightOpen ? <PanelRightClose /> : <PanelRightOpen />}</Button>
                </div>
              </header>
              {!sessionId || detail.isLoading ? (
                <ChatSkeleton />
              ) : detail.isError || !detail.data ? (
                <div className='flex flex-1 items-center justify-center text-sm text-muted-foreground'>
                  对话读取失败，请稍后重试。
                </div>
              ) : (
                <StudioChat
                  key={sessionId}
                  sessionId={sessionId}
                  messages={detail.data.messages}
                  transcript={detail.data.transcript}
                  models={models.data ?? []}
                  modelConfigId={
                    modelConfigId ?? detail.data.session.model_config_id
                  }
                  permissionMode={sessionPermissionMode}
                  skills={skills.data ?? []}
                  assets={detail.data.assets}
                  selectedSkillIds={selectedSkillIds}
				  selectedAssets={selectedAssets}
                  onModelChange={setModelConfigId}
                  onPermissionChange={(mode) => {
                    setPermissionMode(mode)
                    if (sessionId) setActiveSessionId(sessionId)
                  }}
                  onSkillChange={setSelectedSkillIds}
                  onAssetChange={setSelectedAssets}
				  onImportLibraryAsset={(selection) => importLibraryAsset.mutateAsync(selection)}
				  onRunFinished={() => {
					  void queryClient.invalidateQueries({ queryKey: ['studio', 'session', sessionId] })
				  }}
                />
              )}
            </main>
            {rightOpen ? (
              <aside className='hidden w-[42%] max-w-2xl min-w-80 shrink-0 border-s bg-muted/20 xl:flex xl:flex-col'>
                <Tabs defaultValue='flow' className='min-h-0 flex-1 gap-0'>
                  <div className='flex h-16 items-center px-4'>
                    <TabsList>
                      <TabsTrigger value='flow'>资产路线</TabsTrigger>
                      <TabsTrigger value='assets'>
                        Session 资产{' '}
                        <span className='text-xs text-muted-foreground'>
                          {detail.data?.assets.length ?? 0}
                        </span>
                      </TabsTrigger>
                    </TabsList>
                  </div>
                  <TabsContent value='flow' className='m-0 min-h-0'>
                    <StudioFlow
                      nodes={detail.data?.flow.nodes ?? []}
                      edges={detail.data?.flow.edges ?? []}
                      onNodeCreate={(input) => createFlowNode.mutateAsync(input)}
                      onNodeDelete={(id) => deleteFlowNode.mutate(id)}
                      onEdgeCreate={(input) => createFlowEdge.mutateAsync(input)}
                      onEdgeDelete={(id) => deleteFlowEdge.mutate(id)}
                      onPositionsChange={(nodes) =>
                        saveFlowPositions.mutate(nodes)
                      }
                    />
                  </TabsContent>
                  <TabsContent value='assets' className='m-0 min-h-0'>
                    <StudioAssets
                      assets={detail.data?.assets ?? []}
                      onSaveToLibrary={(input) => saveAsset.mutateAsync(input)}
                      onCreateTextAsset={(input) =>
                        createTextAsset.mutate(input)
                      }
                      onUpdateTextAsset={(input) => updateTextAsset.mutateAsync(input)}
                      onUploadAsset={(file) => uploadAsset.mutate(file)}
                      uploading={uploadAsset.isPending}
                    />
                  </TabsContent>
                </Tabs>
              </aside>
            ) : null}
          </section>
        </div>
      ) : null}
      <Sheet open={traceOpen} onOpenChange={setTraceOpen}>
        <SheetContent side='right' className='flex w-full max-w-3xl flex-col gap-0 p-0 sm:max-w-3xl'>
          <SheetTitle className='sr-only'>执行 Trace</SheetTitle>
          {sessionId ? <StudioTrace sessionId={sessionId} /> : null}
        </SheetContent>
      </Sheet>
    </div>
  )
}

function ChatSkeleton() {
  return (
    <div className='mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 p-8'>
      <Skeleton className='mt-10 h-20 w-3/4 self-end rounded-2xl' />
      <Skeleton className='h-28 w-4/5 rounded-2xl' />
      <div className='flex-1' />
      <Skeleton className='h-32 w-full rounded-2xl' />
    </div>
  )
}

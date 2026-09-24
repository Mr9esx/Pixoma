import { useCallback, useEffect, useRef, useState } from 'react'
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
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import {
  Sheet,
  SheetContent,
  SheetDescription,
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
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const chatOpenGeneration = useRef(0)
  const autoCreateRequested = useRef(false)
  const pendingCreateRequestID = useRef<string | undefined>(undefined)

  const sessions = useQuery({
    queryKey: ['studio', 'sessions'],
    queryFn: () => listStudioSessions({ limit: 80 }),
    refetchInterval: 2500,
  })
  const models = useQuery({
    queryKey: ['studio', 'models'],
    queryFn: listStudioModels,
  })
  const skills = useQuery({
    queryKey: ['studio', 'skills'],
    queryFn: listStudioSkills,
  })
  const {
    mutate: createSessionMutate,
    isPending: creatingSession,
    isError: createSessionFailed,
  } = useMutation({
    mutationFn: createStudioSession,
    onSuccess: (session) => {
      pendingCreateRequestID.current = undefined
      setActiveSessionId(session.id)
      setPermissionMode(session.permission_mode)
      setView('chat')
      setTraceOpen(false)
      void queryClient.invalidateQueries({ queryKey: ['studio', 'sessions'] })
    },
  })
  const retryCreateSession = useCallback(() => {
    const requestID = pendingCreateRequestID.current ?? crypto.randomUUID()
    pendingCreateRequestID.current = requestID
    createSessionMutate(requestID)
  }, [createSessionMutate])
  const startNewSession = useCallback(() => {
    const requestID = crypto.randomUUID()
    pendingCreateRequestID.current = requestID
    createSessionMutate(requestID)
  }, [createSessionMutate])
  const sessionId = activeSessionId ?? sessions.data?.[0]?.id
  const sessionPermissionMode = activeSessionId
    ? permissionMode
    : (sessions.data?.[0]?.permission_mode ?? permissionMode)

  useEffect(() => {
    if (
      sessions.isLoading ||
      !sessions.data ||
      sessions.data.length > 0 ||
      creatingSession ||
      autoCreateRequested.current
    ) {
      return
    }
    autoCreateRequested.current = true
    retryCreateSession()
  }, [sessions.data, sessions.isLoading, creatingSession, retryCreateSession])

  const detail = useQuery({
    queryKey: ['studio', 'session', sessionId],
    queryFn: () => getStudioSession(sessionId!),
    enabled: Boolean(sessionId) && view === 'chat',
    staleTime: 0,
    refetchOnMount: 'always',
    refetchOnWindowFocus: true,
    refetchInterval: (query) =>
      query.state.data?.workflow_executions?.some(
        (execution) => execution.status === 'submitted'
      )
        ? 2500
        : false,
  })

  const openChatWithFreshDetail = (id: string, onReady: () => void) => {
    const generation = ++chatOpenGeneration.current
    // The chat runtime decides whether to resume from its first history load.
    // Mount it only after the selected session's current run has been fetched.
    void queryClient
      .fetchQuery({
        queryKey: ['studio', 'session', id],
        queryFn: () => getStudioSession(id),
        staleTime: 0,
      })
      .catch(() => undefined)
      .then(() => {
        if (chatOpenGeneration.current === generation) onReady()
      })
  }

  useEffect(() => {
    const reconcile = () => {
      if (document.visibilityState !== 'visible' || !navigator.onLine) return
      void queryClient.invalidateQueries({ queryKey: ['studio', 'sessions'] })
      if (sessionId && view === 'chat') {
        void queryClient.invalidateQueries({
          queryKey: ['studio', 'session', sessionId],
        })
      }
    }
    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible') reconcile()
    }
    const onOnline = () => reconcile()
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('online', onOnline)
    return () => {
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('online', onOnline)
    }
  }, [queryClient, sessionId, view])

  const saveAsset = useMutation({
    mutationFn: ({
      assetId,
      folderId,
    }: {
      assetId: string
      folderId?: string
    }) => saveStudioAssetToLibrary(assetId, folderId),
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
    onSuccess: () =>
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      }),
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
    onNewSession: startNewSession,
    onSelectSession: (id: string) => {
      openChatWithFreshDetail(id, () => {
        setActiveSessionId(id)
        setPermissionMode(
          sessions.data?.find((session) => session.id === id)
            ?.permission_mode ?? 'request_approval'
        )
        setView('chat')
        setTraceOpen(false)
      })
    },
    onViewChange: (nextView: StudioView) => {
      if (nextView === 'chat' && sessionId) {
        openChatWithFreshDetail(sessionId, () => setView('chat'))
      } else {
        ++chatOpenGeneration.current
        setView(nextView)
      }
    },
  }
  const mobileSidebarProps = {
    ...sidebarProps,
    onNewSession: () => {
      setMobileMenuOpen(false)
      sidebarProps.onNewSession()
    },
    onSelectSession: (id: string) => {
      setMobileMenuOpen(false)
      sidebarProps.onSelectSession(id)
    },
    onViewChange: (nextView: StudioView) => {
      setMobileMenuOpen(false)
      sidebarProps.onViewChange(nextView)
    },
  }

  const workbenchTabs = () => (
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
          workflowExecutions={detail.data?.workflow_executions}
          onNodeCreate={(input) => createFlowNode.mutateAsync(input)}
          onNodeDelete={(id) => deleteFlowNode.mutateAsync(id)}
          onEdgeCreate={(input) => createFlowEdge.mutateAsync(input)}
          onEdgeDelete={(id) => deleteFlowEdge.mutateAsync(id)}
          onPositionsChange={(nodes) => saveFlowPositions.mutateAsync(nodes)}
        />
      </TabsContent>
      <TabsContent value='assets' className='m-0 min-h-0'>
        <StudioAssets
          assets={detail.data?.assets ?? []}
          onSaveToLibrary={(input) => saveAsset.mutateAsync(input)}
          onCreateTextAsset={(input) => createTextAsset.mutateAsync(input)}
          onUpdateTextAsset={(input) => updateTextAsset.mutateAsync(input)}
          onUploadAsset={(file) => uploadAsset.mutate(file)}
          uploading={uploadAsset.isPending}
        />
      </TabsContent>
    </Tabs>
  )

  return (
    <div className='flex h-svh min-h-0 w-full overflow-hidden'>
      <div className='hidden lg:flex'>
        <StudioSidebar {...sidebarProps} />
      </div>
      <div className='fixed top-3 left-3 z-30 lg:hidden'>
        <Sheet open={mobileMenuOpen} onOpenChange={setMobileMenuOpen}>
          <SheetTrigger asChild>
            <Button
              variant='outline'
              size='icon'
              className='size-11'
              aria-label='打开 Studio 菜单'
            >
              <Menu />
            </Button>
          </SheetTrigger>
          <SheetContent side='left' className='w-64 bg-muted/30 p-0'>
            <SheetTitle className='sr-only'>Studio 菜单</SheetTitle>
            <SheetDescription className='sr-only'>
              切换对话、资产库和 AI 设置。
            </SheetDescription>
            <StudioSidebar {...mobileSidebarProps} />
          </SheetContent>
        </Sheet>
      </div>

      {view !== 'chat' && createSessionFailed ? (
        <div
          role='alert'
          className='fixed top-4 right-4 z-40 flex max-w-[min(24rem,calc(100vw-2rem))] flex-wrap items-center gap-2 rounded-xl border bg-card p-3 text-sm text-muted-foreground'
        >
          <span>新建对话失败，当前页面未受影响。</span>
          <Button
            variant='outline'
            size='sm'
            className='min-h-11'
            disabled={creatingSession}
            onClick={retryCreateSession}
          >
            重试新建对话
          </Button>
        </div>
      ) : null}

      {view === 'library' ? (
        <StudioLibrary
          onOpenSession={(id) => {
            openChatWithFreshDetail(id, () => {
              setActiveSessionId(id)
              setPermissionMode(
                sessions.data?.find((session) => session.id === id)
                  ?.permission_mode ?? 'request_approval'
              )
              setView('chat')
              setTraceOpen(false)
            })
          }}
        />
      ) : null}
      {view === 'settings' ? <StudioSettings /> : null}
      {view === 'chat' && sessions.isError && !sessions.data ? (
        <main id='main-content' className='min-h-0 min-w-0 flex-1 p-3 sm:p-4'>
          <div className='flex h-full rounded-2xl border bg-card'>
            <Empty>
              <EmptyHeader>
                <EmptyTitle>对话列表读取失败</EmptyTitle>
                <EmptyDescription>检查连接后重试。</EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button
                  variant='outline'
                  className='min-h-11'
                  disabled={sessions.isFetching}
                  onClick={() => void sessions.refetch()}
                >
                  重试读取对话
                </Button>
              </EmptyContent>
            </Empty>
          </div>
        </main>
      ) : null}
      {view === 'chat' && (!sessions.isError || sessions.data) ? (
        <div className='min-h-0 min-w-0 flex-1 p-3 sm:p-4'>
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
                  <Button
                    variant={traceOpen ? 'secondary' : 'ghost'}
                    size='sm'
                    className='min-h-11'
                    onClick={() => {
                      if (traceOpen && sessionId) {
                        openChatWithFreshDetail(sessionId, () =>
                          setTraceOpen(false)
                        )
                      } else {
                        ++chatOpenGeneration.current
                        setTraceOpen(true)
                      }
                    }}
                    aria-pressed={traceOpen}
                  >
                    <ListTree />
                    {traceOpen ? '对话' : '轨迹'}
                  </Button>
                  {!traceOpen ? (
                    <Sheet>
                      <SheetTrigger asChild>
                        <Button
                          variant='ghost'
                          size='icon'
                          className='size-11 xl:hidden'
                          aria-label='打开创作工作台'
                        >
                          <PanelRightOpen />
                        </Button>
                      </SheetTrigger>
                      <SheetContent
                        side='right'
                        className='w-full max-w-none gap-0 bg-card p-0 sm:w-[min(90vw,42rem)] sm:max-w-none'
                      >
                        <SheetTitle className='px-4 pt-4 text-sm'>
                          创作工作台
                        </SheetTitle>
                        <SheetDescription className='sr-only'>
                          查看资产路线与 Session 资产。
                        </SheetDescription>
                        {workbenchTabs()}
                      </SheetContent>
                    </Sheet>
                  ) : null}
                  {!traceOpen ? (
                    <Button
                      variant='ghost'
                      size='icon'
                      className='hidden xl:inline-flex'
                      onClick={() => setRightOpen((open) => !open)}
                      aria-label={rightOpen ? '收起右侧面板' : '展开右侧面板'}
                    >
                      {rightOpen ? <PanelRightClose /> : <PanelRightOpen />}
                    </Button>
                  ) : null}
                </div>
              </header>
              {sessionId && createSessionFailed ? (
                <div
                  role='alert'
                  className='flex flex-wrap items-center gap-2 border-b px-5 py-2 text-sm text-muted-foreground'
                >
                  <span>新建对话失败，当前对话未受影响。</span>
                  <Button
                    variant='outline'
                    size='sm'
                    className='min-h-11'
                    disabled={creatingSession}
                    onClick={retryCreateSession}
                  >
                    重试新建对话
                  </Button>
                </div>
              ) : null}
              {traceOpen && sessionId ? (
                <StudioTrace key={sessionId} sessionId={sessionId} />
              ) : !sessionId && createSessionFailed ? (
                <Empty>
                  <EmptyHeader>
                    <EmptyTitle>新建对话失败</EmptyTitle>
                    <EmptyDescription>检查连接后重试。</EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent>
                    <Button
                      variant='outline'
                      className='min-h-11'
                      disabled={creatingSession}
                      onClick={retryCreateSession}
                    >
                      重试新建对话
                    </Button>
                  </EmptyContent>
                </Empty>
              ) : !sessionId || detail.isLoading ? (
                <ChatSkeleton />
              ) : detail.isError || !detail.data ? (
                <Empty>
                  <EmptyHeader>
                    <EmptyTitle>对话读取失败</EmptyTitle>
                    <EmptyDescription>检查连接后重试。</EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent>
                    <Button
                      variant='outline'
                      className='min-h-11'
                      disabled={detail.isFetching}
                      onClick={() => void detail.refetch()}
                    >
                      重试读取
                    </Button>
                  </EmptyContent>
                </Empty>
              ) : (
                <StudioChat
                  key={sessionId}
                  sessionId={sessionId}
                  messages={detail.data.messages}
                  transcript={detail.data.transcript}
                  latestRun={detail.data.session.latest_run}
                  runProgress={detail.data.run_progress}
                  pendingApprovals={detail.data.pending_approvals}
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
                  onImportLibraryAsset={(selection) =>
                    importLibraryAsset.mutateAsync(selection)
                  }
                  onRunFinished={() => {
                    void queryClient.invalidateQueries({
                      queryKey: ['studio', 'session', sessionId],
                    })
                    void queryClient.invalidateQueries({
                      queryKey: ['studio', 'sessions'],
                    })
                  }}
                  onRuntimeStateChange={() => {
                    void queryClient.invalidateQueries({
                      queryKey: ['studio', 'sessions'],
                    })
                  }}
                />
              )}
            </main>
            {rightOpen && !traceOpen ? (
              <aside className='hidden w-[42%] max-w-2xl min-w-80 shrink-0 border-s bg-muted/20 xl:flex xl:flex-col'>
                {workbenchTabs()}
              </aside>
            ) : null}
          </section>
        </div>
      ) : null}
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

import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMatchRoute, useNavigate, useSearch } from '@tanstack/react-router'
import { ListTree, Menu, PanelRightClose, PanelRightOpen } from 'lucide-react'
import { ApiError } from '@/lib/api/client'
import {
  createStudioTextAsset,
  createStudioFlowEdge,
  createStudioFlowNode,
  deleteStudioFlowEdge,
  deleteStudioFlowNode,
  getStudioSession,
  listStudioModels,
  listStudioSkills,
  saveStudioAssetToLibrary,
  importStudioLibraryAsset,
  uploadStudioAsset,
  updateStudioTextAsset,
  updateStudioFlowNodes,
  type StudioPermissionMode,
  type StudioSessionDetail,
} from '@/lib/api/studio'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { IconButtonTooltip } from '@/components/ui/icon-button-tooltip'
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
import { StudioNewSession } from './studio-new-session'
import { StudioSettings, type SettingSection } from './studio-settings'
import { StudioSidebar, type StudioView } from './studio-sidebar'
import { StudioTraceTabs } from './studio-trace-tabs'
import { StudioWorkbenchBackground } from './studio-workbench-background'

type SelectedAsset = { assetId: string; assetVersionId: string }
const viewedRunsStorageKey = 'studio.viewedRunIds'

export function StudioWorkspace() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const matchRoute = useMatchRoute()
  const search = useSearch({ strict: false })
  const libraryMatch = matchRoute({ to: '/studio/library' })
  const settingsMatch = matchRoute({ to: '/studio/settings/$section' })
  const sessionMatch = matchRoute({
    to: '/studio/sessions/$sessionId',
    fuzzy: true,
  })
  const view: StudioView = libraryMatch
    ? 'library'
    : settingsMatch
      ? 'settings'
      : 'chat'
  const activeSessionId = sessionMatch ? sessionMatch.sessionId : undefined
  const traceOpen = Boolean(
    matchRoute({ to: '/studio/sessions/$sessionId/trace' })
  )
  const panel = search.panel ?? 'flow'
  const matchedSection = settingsMatch ? settingsMatch.section : undefined
  const section = (matchedSection ?? 'models') as SettingSection
  const [rightOpen, setRightOpen] = useState(true)
  const workbenchOpen = rightOpen && !traceOpen
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [workbenchSheetOpen, setWorkbenchSheetOpen] = useState(false)
  const [locateMessage, setLocateMessage] = useState<{
    id: string
    request: number
  }>()
  const [viewedRunIds, setViewedRunIds] = useState<Record<string, string>>(() =>
    JSON.parse(window.localStorage.getItem(viewedRunsStorageKey) ?? '{}')
  )
  const chatOpenGeneration = useRef(0)
  const [newSessionKey, setNewSessionKey] = useState(0)
  const lastSection = useRef<SettingSection>('models')
  const [modelSelection, setModelSelection] = useState<{
    sessionId: string
    modelConfigId: string
  }>()
  const [skillSelection, setSkillSelection] = useState<{
    sessionId: string
    skillIds: string[]
  }>()
  const [assetSelection, setAssetSelection] = useState<{
    sessionId: string
    assets: SelectedAsset[]
  }>()
  const [permissionOverride, setPermissionOverride] = useState<{
    sessionId: string
    mode: StudioPermissionMode
  }>()

  const models = useQuery({
    queryKey: ['studio', 'models'],
    queryFn: listStudioModels,
  })
  const skills = useQuery({
    queryKey: ['studio', 'skills'],
    queryFn: listStudioSkills,
  })
  const startNewSession = (projectId?: string) => {
    ++chatOpenGeneration.current
    setNewSessionKey((current) => current + 1)
    void navigate({ to: '/studio', search: { project: projectId } })
  }
  const sessionId = activeSessionId

  useEffect(() => {
    if (matchedSection) lastSection.current = section
  }, [matchedSection, section])

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
  useEffect(() => {
    window.localStorage.setItem(
      viewedRunsStorageKey,
      JSON.stringify(viewedRunIds)
    )
  }, [viewedRunIds])

  useEffect(() => {
    const viewedSessionId = detail.data?.session.id
    const run = detail.data?.session.latest_run
    if (
      view !== 'chat' ||
      traceOpen ||
      !viewedSessionId ||
      viewedSessionId !== sessionId ||
      run?.status !== 'succeeded' ||
      viewedRunIds[viewedSessionId] === run.id
    ) {
      return
    }
    setViewedRunIds((current) => ({
      ...current,
      [viewedSessionId]: run.id,
    }))
  }, [detail.data, sessionId, traceOpen, view, viewedRunIds])

  const sessionPermissionMode =
    permissionOverride && permissionOverride.sessionId === sessionId
      ? permissionOverride.mode
      : (detail.data?.session.permission_mode ?? 'request_approval')

  const openChatWithFreshDetail = (
    id: string,
    onReady: (sessionDetail?: StudioSessionDetail) => void
  ) => {
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
      .then((sessionDetail) => {
        if (chatOpenGeneration.current === generation) onReady(sessionDetail)
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
      categoryId,
    }: {
      assetId: string
      categoryId?: string
    }) => saveStudioAssetToLibrary(assetId, categoryId),
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
  const currentSessionDetail =
    detail.data?.session.id === sessionId ? detail.data : undefined
  const uploadAsset = useMutation({
    mutationFn: (file: File) => uploadStudioAsset(file, sessionId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['studio', 'session', sessionId],
      })
    },
  })

  const sidebarProps = {
    activeSessionId: sessionId,
    viewedRunIds,
    view,
    onNewSession: startNewSession,
    onSelectSession: (id: string) => {
      openChatWithFreshDetail(id, () => {
        void navigate({
          to: '/studio/sessions/$sessionId',
          params: { sessionId: id },
          search: {},
        })
      })
    },
    onViewChange: (nextView: StudioView) => {
      if (view === 'settings') lastSection.current = section
      if (nextView === 'library') {
        ++chatOpenGeneration.current
        void navigate({ to: '/studio/library', search: {} })
      } else if (nextView === 'settings') {
        ++chatOpenGeneration.current
        void navigate({
          to: '/studio/settings/$section',
          params: {
            section: view === 'settings' ? section : lastSection.current,
          },
          search: {},
        })
      } else {
        startNewSession()
      }
    },
  }
  const mobileSidebarProps = {
    ...sidebarProps,
    onNewSession: (projectId?: string) => {
      setMobileMenuOpen(false)
      sidebarProps.onNewSession(projectId)
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

  const renderWorkbenchContent = (kind: 'flow' | 'assets') => {
    if (!sessionId || (detail.isError && !currentSessionDetail)) return null
    if (!currentSessionDetail) {
      return (
        <div aria-busy='true' className='h-full min-h-0 p-4'>
          <Skeleton className='h-full w-full' />
        </div>
      )
    }
    if (kind === 'flow') {
      return (
        <StudioFlow
          background={false}
          nodes={currentSessionDetail.flow.nodes}
          edges={currentSessionDetail.flow.edges}
          workflowExecutions={currentSessionDetail.workflow_executions}
          onNodeCreate={(input) => createFlowNode.mutateAsync(input)}
          onNodeDelete={(id) => deleteFlowNode.mutateAsync(id)}
          onEdgeCreate={(input) => createFlowEdge.mutateAsync(input)}
          onEdgeDelete={(id) => deleteFlowEdge.mutateAsync(id)}
          onPositionsChange={(nodes) => saveFlowPositions.mutateAsync(nodes)}
        />
      )
    }
    return (
      <StudioAssets
        assets={currentSessionDetail.assets}
        messages={currentSessionDetail.messages}
        onLocateMessage={(id) => {
          setLocateMessage((current) => ({
            id,
            request: (current?.request ?? 0) + 1,
          }))
          setWorkbenchSheetOpen(false)
        }}
        onSaveToLibrary={(input) => saveAsset.mutateAsync(input)}
        onCreateTextAsset={(input) => createTextAsset.mutateAsync(input)}
        onUpdateTextAsset={(input) => updateTextAsset.mutateAsync(input)}
        onUploadAsset={(file) => uploadAsset.mutate(file)}
        uploading={uploadAsset.isPending}
      />
    )
  }

  const workbenchTabs = () => (
    <Tabs defaultValue='flow' className='relative min-h-0 flex-1 gap-0'>
      <StudioWorkbenchBackground />
      <TabsList className='absolute top-3 left-4 z-10'>
        <TabsTrigger value='flow'>制作流程</TabsTrigger>
        <TabsTrigger value='assets'>会话资产</TabsTrigger>
      </TabsList>
      <TabsContent value='flow' className='relative z-0 m-0 min-h-0'>
        {renderWorkbenchContent('flow')}
      </TabsContent>
      <TabsContent value='assets' className='relative z-0 m-0 min-h-0 pt-16'>
        {renderWorkbenchContent('assets')}
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
          <IconButtonTooltip label='打开 Studio 菜单'>
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
          </IconButtonTooltip>
          <SheetContent side='left' className='w-64 bg-muted/30 p-0'>
            <SheetTitle className='sr-only'>Studio 菜单</SheetTitle>
            <SheetDescription className='sr-only'>
              切换对话、资产库和 AI 设置。
            </SheetDescription>
            <StudioSidebar {...mobileSidebarProps} />
          </SheetContent>
        </Sheet>
      </div>

      {view === 'library' ? (
        <StudioLibrary
          onOpenSession={(id, sourceRunId) => {
            openChatWithFreshDetail(id, (sessionDetail) => {
              const sourceMessageId = sourceRunId
                ? sessionDetail?.messages.find(
                    (message) =>
                      message.role === 'user' && message.run_id === sourceRunId
                  )?.id
                : undefined
              if (sourceMessageId) {
                setLocateMessage((current) => ({
                  id: sourceMessageId,
                  request: (current?.request ?? 0) + 1,
                }))
              }
              void navigate({
                to: '/studio/sessions/$sessionId',
                params: { sessionId: id },
                search: {},
              })
            })
          }}
        />
      ) : null}
      {view === 'settings' ? (
        <StudioSettings
          section={section}
          onSessionsCleared={() => {
            ++chatOpenGeneration.current
            setViewedRunIds({})
            setLocateMessage(undefined)
            setModelSelection(undefined)
            setSkillSelection(undefined)
            setAssetSelection(undefined)
            setPermissionOverride(undefined)
          }}
          onSectionChange={(nextSection) => {
            lastSection.current = nextSection
            void navigate({
              to: '/studio/settings/$section',
              params: { section: nextSection },
              search: {},
            })
          }}
        />
      ) : null}
      {view === 'chat' && !sessionId ? (
        <StudioNewSession
          key={`${newSessionKey}:${search.project ?? ''}`}
          models={models.data ?? []}
          skills={skills.data ?? []}
          initialProjectId={search.project}
        />
      ) : null}
      {view === 'chat' && sessionId ? (
        <div className='min-h-0 min-w-0 flex-1 py-3 pr-3 sm:py-4 sm:pr-4'>
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
                    disabled={!sessionId}
                    onClick={() => {
                      if (!sessionId) return
                      if (traceOpen) {
                        openChatWithFreshDetail(sessionId, () => {
                          void navigate({
                            to: '/studio/sessions/$sessionId',
                            params: { sessionId },
                            search: {},
                          })
                        })
                      } else {
                        ++chatOpenGeneration.current
                        void navigate({
                          to: '/studio/sessions/$sessionId/trace',
                          params: { sessionId },
                          search: {},
                        })
                      }
                    }}
                    aria-pressed={traceOpen}
                  >
                    <ListTree />
                    {traceOpen ? '对话' : '轨迹'}
                  </Button>
                  {!traceOpen ? (
                    <Sheet
                      open={workbenchSheetOpen}
                      onOpenChange={setWorkbenchSheetOpen}
                    >
                      <IconButtonTooltip label='打开创作工作台'>
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
                      </IconButtonTooltip>
                      <SheetContent
                        side='right'
                        className='w-full max-w-none gap-0 bg-card p-0 sm:w-[min(90vw,42rem)] sm:max-w-none'
                      >
                        <SheetTitle className='px-4 pt-4 text-sm'>
                          创作工作台
                        </SheetTitle>
                        <SheetDescription className='sr-only'>
                          查看制作流程与会话资产。
                        </SheetDescription>
                        {workbenchTabs()}
                      </SheetContent>
                    </Sheet>
                  ) : null}
                  {!workbenchOpen && !traceOpen ? (
                    <IconButtonTooltip label='展开右侧面板'>
                      <Button
                        variant='ghost'
                        size='icon'
                        className='hidden xl:inline-flex'
                        onClick={() => setRightOpen(true)}
                        aria-label='展开右侧面板'
                      >
                        <PanelRightOpen />
                      </Button>
                    </IconButtonTooltip>
                  ) : null}
                </div>
              </header>
              {detail.error instanceof ApiError &&
              detail.error.status === 404 ? (
                <Empty>
                  <EmptyHeader>
                    <EmptyTitle>对话不存在或已清空</EmptyTitle>
                  </EmptyHeader>
                </Empty>
              ) : traceOpen && sessionId ? (
                <StudioTraceTabs key={sessionId} sessionId={sessionId} />
              ) : detail.isLoading ? (
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
                  pendingClarifications={detail.data.pending_clarifications}
                  models={models.data ?? []}
                  modelConfigId={
                    modelSelection?.sessionId === sessionId
                      ? modelSelection.modelConfigId
                      : detail.data.session.model_config_id
                  }
                  permissionMode={sessionPermissionMode}
                  skills={skills.data ?? []}
                  assets={detail.data.assets}
                  locateMessage={locateMessage}
                  selectedSkillIds={
                    skillSelection?.sessionId === sessionId
                      ? skillSelection.skillIds
                      : []
                  }
                  selectedAssets={
                    assetSelection?.sessionId === sessionId
                      ? assetSelection.assets
                      : []
                  }
                  onModelChange={(modelConfigId) => {
                    setModelSelection({ sessionId, modelConfigId })
                  }}
                  onPermissionChange={(mode) => {
                    if (sessionId) setPermissionOverride({ sessionId, mode })
                  }}
                  onSkillChange={(skillIds) => {
                    setSkillSelection({ sessionId, skillIds })
                  }}
                  onAssetChange={(assets) => {
                    setAssetSelection({ sessionId, assets })
                  }}
                  onImportLibraryAsset={(selection) =>
                    importLibraryAsset.mutateAsync(selection)
                  }
                  onUploadAsset={(file) => uploadAsset.mutateAsync(file)}
                  onRunFinished={() => {
                    void queryClient.invalidateQueries({
                      queryKey: ['studio', 'session', sessionId],
                    })
                    void queryClient.invalidateQueries({
                      queryKey: ['studio', 'sessions'],
                    })
                    void queryClient.invalidateQueries({
                      queryKey: ['studio', 'skills'],
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
            <aside
              data-slot='studio-workbench-track'
              aria-hidden={!workbenchOpen}
              inert={!workbenchOpen}
              className={`hidden shrink-0 overflow-hidden transition-[flex-basis,width,min-width,max-width] duration-300 ease-in-out xl:flex xl:flex-col ${
                workbenchOpen
                  ? 'w-[42%] max-w-2xl min-w-80 basis-[42%]'
                  : 'w-0 max-w-0 min-w-0 basis-0'
              }`}
            >
              <div
                data-slot='studio-workbench-panel'
                className={`flex h-full w-full max-w-2xl min-w-80 flex-col border-s bg-muted/20 transition-transform duration-300 ease-in-out ${
                  workbenchOpen
                    ? 'translate-x-0'
                    : 'pointer-events-none translate-x-full'
                }`}
              >
                <Tabs
                  value={panel}
                  onValueChange={(nextPanel) =>
                    void navigate({
                      to: '/studio/sessions/$sessionId',
                      params: { sessionId: sessionId! },
                      search: {
                        panel: nextPanel === 'assets' ? 'assets' : undefined,
                      },
                    })
                  }
                  className='relative min-h-0 min-w-80 flex-1 gap-0'
                >
                  <StudioWorkbenchBackground />
                  <TabsList className='absolute top-3 left-4 z-10'>
                    <TabsTrigger value='flow' disabled={!sessionId}>
                      制作流程
                    </TabsTrigger>
                    <TabsTrigger value='assets' disabled={!sessionId}>
                      会话资产
                    </TabsTrigger>
                  </TabsList>
                  <IconButtonTooltip label='收起右侧面板'>
                    <Button
                      variant='ghost'
                      size='icon'
                      className='absolute top-3 right-5 z-10 hidden xl:inline-flex'
                      onClick={() => setRightOpen(false)}
                      aria-label='收起右侧面板'
                    >
                      <PanelRightClose />
                    </Button>
                  </IconButtonTooltip>
                  <TabsContent
                    value='flow'
                    className='relative z-0 m-0 min-h-0'
                  >
                    {renderWorkbenchContent('flow')}
                  </TabsContent>
                  <TabsContent
                    value='assets'
                    className='relative z-0 m-0 min-h-0 pt-16'
                  >
                    {renderWorkbenchContent('assets')}
                  </TabsContent>
                </Tabs>
              </div>
            </aside>
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

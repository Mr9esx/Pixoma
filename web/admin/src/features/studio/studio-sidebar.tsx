import { useEffect, useRef, useState, type ReactNode } from 'react'
import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  Library,
  MessageCircle,
  MessageSquarePlus,
  MoreHorizontal,
  Plus,
  Settings2,
  Workflow,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  deleteStudioProject,
  listStudioLibraryProjects,
  listStudioProjects,
  listStudioSessions,
  moveStudioSessionToProject,
  type StudioProject,
  type StudioSession,
} from '@/lib/api/studio'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { AppTitle } from '@/components/layout/app-title'
import { NavUser } from '@/components/layout/nav-user'
import { StatusDot } from '@/components/status-dot'
import { StudioProjectDialog } from './studio-project-dialog'
import {
  readProjectExpanded,
  writeProjectExpanded,
} from './studio-project-expanded-state'
import {
  isActiveStudioSessionItem,
  type StudioSessionListSelection,
  type StudioSessionListSource,
} from './studio-sidebar-session-location'
import styles from './studio-sidebar.module.css'

export type StudioView = 'chat' | 'library' | 'settings'

function SessionTitle({ title }: { title: string }) {
  const [isScrolling, setIsScrolling] = useState(false)
  const titleRef = useRef<HTMLSpanElement>(null)

  return (
    <span
      className={styles.titleViewport}
      data-scrolling={isScrolling}
      onPointerEnter={(event) => {
        if (event.pointerType !== 'mouse') return
        if (window.matchMedia('(prefers-reduced-motion: reduce)').matches)
          return

        const titleElement = titleRef.current
        setIsScrolling(
          titleElement !== null &&
            titleElement.scrollWidth > titleElement.clientWidth + 1
        )
      }}
      onPointerLeave={() => setIsScrolling(false)}
    >
      {isScrolling ? (
        <span className={styles.marqueeTrack}>
          <span className={styles.marqueeSegment}>
            <span>{title}</span>
            <span aria-hidden='true' className={styles.marqueeGap} />
          </span>
          <span aria-hidden='true' className={styles.marqueeSegment}>
            <span>{title}</span>
            <span className={styles.marqueeGap} />
          </span>
        </span>
      ) : (
        <span ref={titleRef} className={styles.truncatedTitle}>
          {title}
        </span>
      )}
    </span>
  )
}

type Props = {
  activeSessionId?: string
  viewedRunIds: Record<string, string>
  view: StudioView
  onNewSession: (projectId?: string) => void
  onSelectSession: (id: string) => void
  onViewChange: (view: StudioView) => void
}

function sessionStatus(session: StudioSession, viewedRunId?: string) {
  switch (session.latest_run?.status) {
    case 'queued':
    case 'running':
      return { state: 'active' as const, label: '执行中', pulse: true }
    case 'waiting_approval':
      return { state: 'warn' as const, label: '等待你的操作', pulse: false }
    case 'waiting_clarification':
      return { state: 'warn' as const, label: '等待你的回答', pulse: false }
    case 'failed':
    case 'cancelled':
      return { state: 'warn' as const, label: '本轮未完成', pulse: false }
    case 'succeeded':
      return session.latest_run.id === viewedRunId
        ? null
        : { state: 'ok' as const, label: '本轮已完成', pulse: false }
    default:
      return null
  }
}

export function StudioSidebar({
  activeSessionId,
  viewedRunIds,
  view,
  onNewSession,
  onSelectSession,
  onViewChange,
}: Props) {
  const queryClient = useQueryClient()
  const [projectsExpanded, setProjectsExpanded] = useState(true)
  const [recentExpanded, setRecentExpanded] = useState(true)
  const projects = useQuery({
    queryKey: ['studio', 'projects'],
    queryFn: listStudioProjects,
  })
  const recent = useInfiniteQuery({
    queryKey: ['studio', 'sessions', 'recent'],
    queryFn: ({ pageParam }) =>
      listStudioSessions({ limit: 30, offset: pageParam }),
    initialPageParam: 0,
    getNextPageParam: (page, pages) =>
      page.length === 30 ? pages.length * 30 : undefined,
    enabled: recentExpanded,
    refetchInterval: recentExpanded ? 5000 : false,
  })
  const {
    data: recentData,
    hasNextPage: hasMoreRecent,
    isFetchingNextPage: isFetchingMoreRecent,
    fetchNextPage: fetchMoreRecent,
  } = recent
  const recentScrollRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const element = recentScrollRef.current
    if (
      recentExpanded &&
      element &&
      element.scrollHeight <= element.clientHeight &&
      hasMoreRecent &&
      !isFetchingMoreRecent
    )
      void fetchMoreRecent()
  }, [
    recentExpanded,
    recentData?.pages.length,
    hasMoreRecent,
    isFetchingMoreRecent,
    fetchMoreRecent,
  ])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingProject, setEditingProject] = useState<StudioProject>()
  const [deletingProject, setDeletingProject] = useState<StudioProject>()
  const [openSessionMenuKey, setOpenSessionMenuKey] = useState<string>()
  const [sessionSelection, setSessionSelection] =
    useState<StudioSessionListSelection>()
  const deleteCounts = useQuery({
    queryKey: ['studio', 'project-delete-counts', deletingProject?.id],
    queryFn: async () => {
      const projectId = deletingProject!.id
      const libraryProjects = await listStudioLibraryProjects()
      const project = libraryProjects.find((item) => item.id === projectId)
      if (!project) throw new Error('项目资产数量读取失败')
      let sessionCount = 0
      for (;;) {
        const page = await listStudioSessions({ project_id: projectId, limit: 100, offset: sessionCount })
        sessionCount += page.length
        if (page.length < 100) break
      }
      return { assetCount: project.asset_count, sessionCount }
    },
    enabled: Boolean(deletingProject),
    staleTime: 0,
    refetchOnMount: 'always',
  })
  const removeProject = useMutation({
    mutationFn: (id: string) => deleteStudioProject(id),
    onSuccess: async (_result, projectId) => {
      writeProjectExpanded(projectId, false)
      setDeletingProject(undefined)
      await queryClient.invalidateQueries({ queryKey: ['studio', 'projects'] })
      await queryClient.invalidateQueries({ queryKey: ['studio', 'sessions'] })
      await queryClient.invalidateQueries({ queryKey: ['studio', 'session'] })
      await queryClient.invalidateQueries({ queryKey: ['studio', 'library'] })
    },
  })
  const moveSession = useMutation({
    mutationFn: ({
      sessionId,
      projectId,
    }: {
      sessionId: string
      projectId: string
    }) => moveStudioSessionToProject(sessionId, projectId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['studio', 'sessions'] })
      void queryClient.invalidateQueries({ queryKey: ['studio', 'session'] })
    },
    onError: (error) => toast.error(error.message),
  })

  const sessionItem = (session: StudioSession, isProjectSession = false) => {
    const status = sessionStatus(session, viewedRunIds[session.id])
    const workflowRunning = (session.active_workflow_count ?? 0) > 0
    const source: StudioSessionListSource = isProjectSession
      ? 'project'
      : 'recent'
    const menuKey = `${source}:${session.id}`
    const availableProjects = (projects.data ?? []).filter(
      (project) => project.id !== session.project_id
    )
    return (
      <SidebarMenuItem
        key={session.id}
        className='group/session relative w-full min-w-0'
      >
        <SidebarMenuButton
          className={
            isProjectSession
              ? `w-full min-w-0 ps-8 ${workflowRunning && status ? 'pe-12' : 'pe-9'}`
              : `w-full min-w-0 ${workflowRunning && status ? 'pe-12' : 'pe-9'}`
          }
          isActive={
            view === 'chat' &&
            isActiveStudioSessionItem({
              activeSessionId,
              selection: sessionSelection,
              sessionId: session.id,
              sessionProjectId: session.project_id,
              source,
            })
          }
          onClick={() => {
            setSessionSelection({ sessionId: session.id, source })
            onSelectSession(session.id)
          }}
        >
          <SessionTitle title={session.title} />
        </SidebarMenuButton>
        {(status || workflowRunning) && openSessionMenuKey !== menuKey ? (
          <span className='pointer-events-none absolute end-1 top-1/2 z-10 flex h-7 -translate-y-1/2 items-center gap-1.5 transition-opacity group-focus-within/session:opacity-0 group-hover/session:opacity-0 [@media(hover:none)]:opacity-0'>
            {workflowRunning ? <span role='img' aria-label={`${session.active_workflow_count} 个工作流任务执行中`} title={`${session.active_workflow_count} 个工作流任务执行中`} className='inline-flex size-5 items-center justify-center text-primary motion-safe:animate-pulse'><Workflow aria-hidden='true' className='size-3.5' /></span> : null}
            {status ? <StatusDot {...status} /> : null}
          </span>
        ) : null}
        <DropdownMenu
          open={openSessionMenuKey === menuKey}
          onOpenChange={(open) =>
            setOpenSessionMenuKey((current) =>
              open ? menuKey : current === menuKey ? undefined : current
            )
          }
        >
          <DropdownMenuTrigger asChild>
            <Button
              size='icon'
              variant='ghost'
              className='pointer-events-none absolute end-0 top-1/2 z-10 size-7 -translate-y-1/2 text-muted-foreground/80 opacity-0 transition-opacity group-focus-within/session:pointer-events-auto group-focus-within/session:opacity-100 group-hover/session:pointer-events-auto group-hover/session:opacity-100 hover:text-muted-foreground data-[state=open]:pointer-events-auto data-[state=open]:opacity-100 [@media(hover:none)]:pointer-events-auto [@media(hover:none)]:opacity-100'
              aria-label={`${session.title}的更多操作`}
            >
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align='end'>
            {isProjectSession && session.project_id ? (
              <DropdownMenuItem
                onSelect={() =>
                  moveSession.mutate({ sessionId: session.id, projectId: '' })
                }
              >
                移出项目
              </DropdownMenuItem>
            ) : null}
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>移至项目</DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                {availableProjects.length > 0 ? (
                  availableProjects.map((project) => (
                    <DropdownMenuItem
                      key={project.id}
                      onSelect={() =>
                        moveSession.mutate({
                          sessionId: session.id,
                          projectId: project.id,
                        })
                      }
                    >
                      {project.name}
                    </DropdownMenuItem>
                  ))
                ) : (
                  <DropdownMenuItem disabled>暂无可选项目</DropdownMenuItem>
                )}
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    )
  }

  const ProjectsChevron = projectsExpanded ? ChevronDown : ChevronRight
  const RecentChevron = recentExpanded ? ChevronDown : ChevronRight

  return (
    <Sidebar collapsible='none' className='p-2'>
      <SidebarHeader>
        <AppTitle showToggle={false} />
      </SidebarHeader>

      <SidebarContent className='gap-0 overflow-hidden'>
        <SidebarGroup className='shrink-0 px-2 py-1'>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild tooltip='返回后台'>
                  <Link to='/'>
                    <ArrowLeft />
                    <span>返回后台</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  isActive={view === 'chat'}
                  onClick={() => onViewChange('chat')}
                >
                  <MessageCircle />
                  <span>对话</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  isActive={view === 'library'}
                  onClick={() => onViewChange('library')}
                >
                  <Library />
                  <span>资产库</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton
                  isActive={view === 'settings'}
                  onClick={() => onViewChange('settings')}
                >
                  <Settings2 />
                  <span>AI 设置</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup
          className={`${styles.section} px-2 py-1`}
          data-expanded={projectsExpanded}
        >
          <SidebarGroupLabel className='group/section-label justify-between pe-0'>
            <Button
              variant='ghost'
              size='sm'
              className='h-8 min-w-0 flex-1 justify-start gap-1 px-0 text-sm font-normal text-muted-foreground/80 hover:bg-transparent hover:text-muted-foreground/80 has-[>svg]:px-0'
              aria-expanded={projectsExpanded}
              aria-label={projectsExpanded ? '折叠项目' : '展开项目'}
              onClick={() => setProjectsExpanded((value) => !value)}
            >
              <span>项目</span>
              <ProjectsChevron className='size-4 text-muted-foreground/80 opacity-0 group-focus-within/section-label:opacity-100 group-hover/section-label:opacity-100 [@media(hover:none)]:opacity-100' />
            </Button>
            <Button
              size='icon'
              variant='ghost'
              className='size-7 p-0 text-muted-foreground/80 opacity-0 transition-opacity group-focus-within/section-label:opacity-100 group-hover/section-label:opacity-100 hover:text-muted-foreground [@media(hover:none)]:opacity-100'
              aria-label='新建项目'
              onClick={() => {
                setEditingProject(undefined)
                setDialogOpen(true)
              }}
            >
              <Plus />
            </Button>
          </SidebarGroupLabel>
          <SidebarGroupContent
            className={`${styles.sectionContent} flex min-h-0 w-full min-w-0 flex-col self-stretch`}
            aria-hidden={!projectsExpanded}
            inert={!projectsExpanded}
          >
            <div className='studio-scrollbar min-h-0 w-full min-w-0 flex-auto overflow-x-hidden overflow-y-auto'>
              <SidebarMenu className='pb-2'>
                {projects.data?.map((project) => (
                  <ProjectSessions
                    key={project.id}
                    project={project}
                    onNewSession={onNewSession}
                    onRename={() => {
                      setEditingProject(project)
                      setDialogOpen(true)
                    }}
                    onDelete={() => setDeletingProject(project)}
                    renderSession={(session) => sessionItem(session, true)}
                  />
                ))}
                {projects.data?.length === 0 ? (
                  <p className='px-2 py-3 text-sm text-muted-foreground'>
                    暂无项目
                  </p>
                ) : null}
                {projects.isError ? (
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => void projects.refetch()}
                  >
                    重试读取项目
                  </Button>
                ) : null}
              </SidebarMenu>
            </div>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup
          className={`${styles.section} ${styles.recentSection} px-2 py-1`}
          data-expanded={recentExpanded}
        >
          <SidebarGroupLabel className='group/section-label justify-between pe-0'>
            <Button
              variant='ghost'
              size='sm'
              className='h-8 min-w-0 flex-1 justify-start gap-1 px-0 text-sm font-normal text-muted-foreground/80 hover:bg-transparent hover:text-muted-foreground/80 has-[>svg]:px-0'
              aria-expanded={recentExpanded}
              aria-label={recentExpanded ? '折叠最近对话' : '展开最近对话'}
              onClick={() => setRecentExpanded((value) => !value)}
            >
              <span>最近对话</span>
              <RecentChevron className='size-4 text-muted-foreground/80 opacity-0 group-focus-within/section-label:opacity-100 group-hover/section-label:opacity-100 [@media(hover:none)]:opacity-100' />
            </Button>
            <Button
              size='icon'
              variant='ghost'
              className='size-7 p-0 text-muted-foreground/80 opacity-0 transition-opacity group-focus-within/section-label:opacity-100 group-hover/section-label:opacity-100 hover:text-muted-foreground [@media(hover:none)]:opacity-100'
              aria-label='新建对话'
              onClick={() => onNewSession()}
            >
              <MessageSquarePlus />
            </Button>
          </SidebarGroupLabel>
          <SidebarGroupContent
            className={`${styles.sectionContent} flex min-h-0 flex-col`}
            aria-hidden={!recentExpanded}
            inert={!recentExpanded}
          >
            <div
              ref={recentScrollRef}
              className='studio-scrollbar min-h-0 flex-auto overflow-x-hidden overflow-y-auto'
              onScroll={(event) => {
                const element = event.currentTarget
                if (
                  element.scrollHeight -
                    element.scrollTop -
                    element.clientHeight <
                    120 &&
                  recent.hasNextPage &&
                  !recent.isFetchingNextPage
                )
                  void recent.fetchNextPage()
              }}
            >
              <SidebarMenu className='pb-2'>
                {recent.data?.pages
                  .flat()
                  .map((session) => sessionItem(session))}
                {recent.data?.pages[0]?.length === 0 ? (
                  <p className='px-2 py-3 text-sm text-muted-foreground'>
                    暂无对话
                  </p>
                ) : null}
                {recent.isError ? (
                  <Button
                    variant='ghost'
                    size='sm'
                    onClick={() => void recent.refetch()}
                  >
                    重试读取对话
                  </Button>
                ) : null}
                {recent.isFetchingNextPage ? (
                  <p className='px-2 py-2 text-xs text-muted-foreground'>
                    加载中…
                  </p>
                ) : null}
              </SidebarMenu>
            </div>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      {dialogOpen ? (
        <StudioProjectDialog
          open
          project={editingProject}
          onOpenChange={setDialogOpen}
        />
      ) : null}
      <AlertDialog
        open={Boolean(deletingProject)}
        onOpenChange={(open) => {
          if (!open) setDeletingProject(undefined)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              删除项目「{deletingProject?.name}」？
            </AlertDialogTitle>
            <AlertDialogDescription>
              {deleteCounts.isPending || deleteCounts.isFetching ? '正在读取项目数量…' : deleteCounts.isError ? '项目数量读取失败' : `项目内有 ${deleteCounts.data.assetCount} 项资产、${deleteCounts.data.sessionCount} 个对话。删除后将移至未归属项目。`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {deleteCounts.isError ? <Button variant='outline' onClick={() => void deleteCounts.refetch()}>重试读取数量</Button> : null}
          {removeProject.isError ? (
            <p role='alert' className='text-sm text-destructive'>
              {removeProject.error.message}
            </p>
          ) : null}
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              disabled={removeProject.isPending || deleteCounts.isPending || deleteCounts.isFetching || deleteCounts.isError}
              onClick={(event) => {
                event.preventDefault()
                if (deletingProject) removeProject.mutate(deletingProject.id)
              }}
            >
              删除项目
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <SidebarFooter>
        <NavUser studio />
      </SidebarFooter>
    </Sidebar>
  )
}

function ProjectSessions({
  project,
  onNewSession,
  onRename,
  onDelete,
  renderSession,
}: {
  project: StudioProject
  onNewSession: (projectId?: string) => void
  onRename: () => void
  onDelete: () => void
  renderSession: (session: StudioSession) => ReactNode
}) {
  const [expanded, setExpanded] = useState(() =>
    readProjectExpanded(project.id)
  )
  const sessions = useInfiniteQuery({
    queryKey: ['studio', 'sessions', project.id],
    queryFn: ({ pageParam }) =>
      listStudioSessions({
        project_id: project.id,
        limit: 6,
        offset: pageParam,
      }),
    initialPageParam: 0,
    getNextPageParam: (page, pages) =>
      page.length > 5 ? pages.length * 5 : undefined,
    enabled: expanded,
    refetchInterval: expanded ? 5000 : false,
  })
  return (
    <SidebarMenuItem className='min-w-0'>
      <div className='group/project flex min-w-0 items-center rounded-md hover:bg-sidebar-accent'>
        <Button
          variant='ghost'
          className='h-8 min-w-0 flex-1 justify-start gap-2 pr-1! pl-2! font-normal'
          aria-expanded={expanded}
          onClick={() => {
            const next = !expanded
            writeProjectExpanded(project.id, next)
            setExpanded(next)
          }}
        >
          {expanded ? <FolderOpen className='size-4 shrink-0 text-muted-foreground/80' /> : <Folder className='size-4 shrink-0 text-muted-foreground/80' />}
          <span className='truncate'>{project.name}</span>
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size='icon'
              variant='ghost'
              className='size-7 shrink-0 p-0 text-muted-foreground/80 opacity-0 transition-opacity group-focus-within/project:opacity-100 group-hover/project:opacity-100 hover:text-muted-foreground data-[state=open]:opacity-100 [@media(hover:none)]:opacity-100'
              aria-label={`${project.name}的更多操作`}
            >
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align='end'>
            <DropdownMenuItem onSelect={onRename}>重命名</DropdownMenuItem>
            <DropdownMenuItem variant='destructive' onSelect={onDelete}>
              删除项目
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          size='icon'
          variant='ghost'
          className='size-7 shrink-0 p-0 text-muted-foreground/80 opacity-0 transition-opacity group-focus-within/project:opacity-100 group-hover/project:opacity-100 hover:text-muted-foreground [@media(hover:none)]:opacity-100'
          aria-label={`在${project.name}中新建对话`}
          onClick={() => onNewSession(project.id)}
        >
          <MessageSquarePlus />
        </Button>
      </div>
      <div
        className={styles.projectSessions}
        data-expanded={expanded}
        aria-hidden={!expanded}
        inert={!expanded}
      >
        <div className={styles.projectSessionsInner}>
          <SidebarMenu>
            {sessions.data?.pages
              .flatMap((page) => page.slice(0, 5))
              .map(renderSession)}
          </SidebarMenu>
          {sessions.data?.pages[0]?.length === 0 ? (
            <p className='px-2 py-2 text-sm text-muted-foreground'>暂无对话</p>
          ) : null}
          {sessions.hasNextPage ? (
            <Button
              variant='ghost'
              size='sm'
              className='w-full justify-start text-muted-foreground'
              disabled={sessions.isFetchingNextPage}
              onClick={() => void sessions.fetchNextPage()}
            >
              展开显示
            </Button>
          ) : null}
          {sessions.isError ? (
            <Button
              variant='ghost'
              size='sm'
              onClick={() => void sessions.refetch()}
            >
              重试读取对话
            </Button>
          ) : null}
        </div>
      </div>
    </SidebarMenuItem>
  )
}

import { useRef, useState } from 'react'
import { Link } from '@tanstack/react-router'
import {
  ArrowLeft,
  Library,
  MessageCircle,
  MessageSquarePlus,
  Settings2,
} from 'lucide-react'
import type { StudioSession } from '@/lib/api/studio'
import { Button } from '@/components/ui/button'
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
  sessions: StudioSession[]
  activeSessionId?: string
  viewedRunIds: Record<string, string>
  view: StudioView
  onNewSession: () => void
  onSelectSession: (id: string) => void
  onViewChange: (view: StudioView) => void
  creating?: boolean
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
  sessions,
  activeSessionId,
  viewedRunIds,
  view,
  onNewSession,
  onSelectSession,
  onViewChange,
  creating,
}: Props) {
  return (
    <Sidebar collapsible='none' className='p-2'>
      <SidebarHeader>
        <AppTitle showToggle={false} />
      </SidebarHeader>

      <SidebarContent className='gap-1'>
        <SidebarGroup className='px-2 py-1'>
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

        <SidebarGroup className='min-h-0 flex-1 px-2 py-1'>
          <SidebarGroupLabel>最近对话</SidebarGroupLabel>
          <SidebarGroupContent className='flex min-h-0 w-full min-w-0 flex-1 flex-col self-stretch'>
            <div className='studio-scrollbar min-h-0 w-full min-w-0 flex-1 overflow-x-hidden overflow-y-auto'>
              <SidebarMenu className='pb-2'>
                {sessions.length === 0 ? (
                  <p className='px-2 py-3 text-xs leading-5 text-muted-foreground'>
                    暂无对话
                  </p>
                ) : (
                  sessions.map((session) => {
                    const status = sessionStatus(
                      session,
                      viewedRunIds[session.id]
                    )
                    return (
                      <SidebarMenuItem
                        key={session.id}
                        className='w-full min-w-0'
                      >
                        <SidebarMenuButton
                          className='min-w-0'
                          isActive={
                            view === 'chat' && activeSessionId === session.id
                          }
                          onClick={() => onSelectSession(session.id)}
                        >
                          <SessionTitle title={session.title} />
                          {status ? (
                            <span className='ms-auto flex shrink-0 items-center'>
                              <StatusDot {...status} />
                            </span>
                          ) : null}
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    )
                  })
                )}
              </SidebarMenu>
            </div>
            <Button
              className='mt-2 w-full'
              onClick={onNewSession}
              disabled={creating}
            >
              <MessageSquarePlus />
              <span>新建对话</span>
            </Button>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
    </Sidebar>
  )
}

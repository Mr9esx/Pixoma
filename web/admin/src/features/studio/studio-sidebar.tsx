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
import { StatusDot } from '@/components/status-dot'
import { ScrollArea } from '@/components/ui/scroll-area'
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

export type StudioView = 'chat' | 'library' | 'settings'

type Props = {
  sessions: StudioSession[]
  activeSessionId?: string
  view: StudioView
  onNewSession: () => void
  onSelectSession: (id: string) => void
  onViewChange: (view: StudioView) => void
  creating?: boolean
}

function sessionStatus(session: StudioSession) {
  switch (session.latest_run?.status) {
    case 'queued':
    case 'running':
      return { state: 'active' as const, label: '执行中', pulse: true }
    case 'waiting_approval':
      return { state: 'warn' as const, label: '等待你的操作', pulse: false }
    case 'failed':
    case 'cancelled':
      return { state: 'warn' as const, label: '本轮未完成', pulse: false }
    case 'succeeded':
      return { state: 'ok' as const, label: '本轮已完成', pulse: false }
    default:
      return null
  }
}

export function StudioSidebar({
  sessions,
  activeSessionId,
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

        <SidebarGroup className='min-h-0 flex-1 ps-2 pe-0 py-1'>
          <SidebarGroupLabel>最近对话</SidebarGroupLabel>
          <SidebarGroupContent className='flex min-h-0 w-full min-w-0 flex-1 flex-col self-stretch'>
            <ScrollArea className='min-h-0 w-full min-w-0 flex-1 self-stretch'>
              <SidebarMenu className='w-full min-w-0 self-stretch pb-2'>
                {sessions.length === 0 ? (
                  <p className='px-2 py-3 text-xs leading-5 text-muted-foreground'>
                    开始一次对话后，会自动保存在这里
                  </p>
                ) : (
                  sessions.map((session) => (
                    <SidebarMenuItem key={session.id} className='min-w-0'>
                      <SidebarMenuButton
                        className='min-w-0'
                        isActive={
                          view === 'chat' && activeSessionId === session.id
                        }
                        onClick={() => onSelectSession(session.id)}
                      >
                        <span className='min-w-0 flex-1 truncate'>
                          {session.title}
                        </span>
                        {sessionStatus(session) ? (
                          <div className='ms-auto flex shrink-0 items-center'>
                            <StatusDot {...sessionStatus(session)!} />
                          </div>
                        ) : null}
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))
                )}
              </SidebarMenu>
            </ScrollArea>
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

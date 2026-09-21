import { Link } from '@tanstack/react-router'
import {
  ArrowLeft,
  Library,
  MessageCircle,
  MessageSquarePlus,
  Settings2,
} from 'lucide-react'
import type { StudioSession } from '@/lib/api/studio'
import { AppTitle } from '@/components/layout/app-title'
import { NavUser } from '@/components/layout/nav-user'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

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
    <Sidebar collapsible='none'>
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
          <SidebarGroupLabel className='pr-8'>最近对话</SidebarGroupLabel>
          <SidebarGroupAction
            aria-label='新对话'
            title='新对话'
            className='inset-e-0 top-1.5'
            onClick={onNewSession}
            disabled={creating}
          >
            <MessageSquarePlus />
          </SidebarGroupAction>
          <SidebarGroupContent className='min-h-0 flex-1'>
            <ScrollArea className='h-full'>
              <SidebarMenu className='pb-2'>
                {sessions.length === 0 ? (
                  <p className='px-2 py-3 text-xs leading-5 text-muted-foreground'>
                    开始一次对话后，会自动保存在这里
                  </p>
                ) : (
                  sessions.map((session) => (
                    <SidebarMenuItem key={session.id}>
                      <SidebarMenuButton
                        isActive={
                          view === 'chat' && activeSessionId === session.id
                        }
                        onClick={() => onSelectSession(session.id)}
                      >
                        <MessageCircle />
                        <span>{session.title}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))
                )}
              </SidebarMenu>
            </ScrollArea>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
    </Sidebar>
  )
}

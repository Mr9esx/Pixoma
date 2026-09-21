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
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
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
          <div className='flex h-8 items-center gap-1 px-2'>
            <SidebarGroupLabel className='h-auto flex-1 px-0'>
              最近对话
            </SidebarGroupLabel>
            <Button
              variant='ghost'
              size='sm'
              className='h-7 gap-1 px-2'
              onClick={onNewSession}
              disabled={creating}
            >
              <MessageSquarePlus />
              <span>新建对话</span>
            </Button>
          </div>
          <SidebarGroupContent className='flex min-h-0 flex-1 flex-col'>
            <ScrollArea className='min-h-0 flex-1'>
              <SidebarMenuSub className='mx-0 translate-x-0 border-s-0 px-0 py-1'>
                {sessions.length === 0 ? (
                  <p className='px-2 py-3 text-xs leading-5 text-muted-foreground'>
                    开始一次对话后，会自动保存在这里
                  </p>
                ) : (
                  sessions.map((session) => (
                    <SidebarMenuSubItem key={session.id}>
                      <SidebarMenuSubButton
                        asChild
                        isActive={
                          view === 'chat' && activeSessionId === session.id
                        }
                      >
                        <button
                          type='button'
                          onClick={() => onSelectSession(session.id)}
                        >
                          <span>{session.title}</span>
                        </button>
                      </SidebarMenuSubButton>
                    </SidebarMenuSubItem>
                  ))
                )}
              </SidebarMenuSub>
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

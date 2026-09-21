import { Link } from '@tanstack/react-router'
import {
  Library,
  MessageCircle,
  MessageSquarePlus,
  Settings2,
  Sparkles,
} from 'lucide-react'
import type { StudioSession } from '@/lib/api/studio'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
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
    <aside className='flex h-full w-64 shrink-0 flex-col bg-sidebar text-sidebar-foreground'>
      <SidebarHeader className='p-2'>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip='返回后台'>
              <Link to='/'>
                <Sparkles />
                <span>创作 Studio</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent className='gap-2 px-2'>
        <SidebarGroup className='p-0'>
          <SidebarGroupContent>
            <SidebarMenu>
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

        <SidebarGroup className='min-h-0 flex-1 p-0'>
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

      <SidebarFooter className='p-2'>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton aria-label='当前用户：管理员'>
              <Avatar className='size-6 shrink-0 rounded-md'>
                <AvatarFallback>管</AvatarFallback>
              </Avatar>
              <span>管理员</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </aside>
  )
}

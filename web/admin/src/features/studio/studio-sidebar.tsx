import { Link } from '@tanstack/react-router'
import {
  ArrowLeft,
  BookOpen,
  Library,
  MessageSquarePlus,
  Settings2,
  Sparkles,
} from 'lucide-react'
import type { StudioSession } from '@/lib/api/studio'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
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
      <SidebarHeader className='gap-3 px-3 py-3'>
        <div className='flex items-center gap-2'>
          <Button variant='ghost' size='icon' asChild aria-label='返回后台'>
            <Link to='/'>
              <ArrowLeft />
            </Link>
          </Button>
          <div className='flex min-w-0 items-center gap-2'>
            <span className='flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground'>
              <Sparkles className='size-4' />
            </span>
            <div className='min-w-0'>
              <p className='truncate text-sm font-semibold'>创作 Studio</p>
              <p className='truncate text-xs text-muted-foreground'>让想法成为作品</p>
            </div>
          </div>
        </div>
        <Button className='h-10 w-full justify-start' onClick={onNewSession} disabled={creating}>
          <MessageSquarePlus />
          {creating ? '正在创建…' : '新对话'}
        </Button>
      </SidebarHeader>

      <SidebarContent className='gap-3'>
        <SidebarGroup className='px-2 py-0'>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={view === 'library'} onClick={() => onViewChange('library')}>
                  <Library />
                  <span>资产库</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={view === 'settings'} onClick={() => onViewChange('settings')}>
                  <Settings2 />
                  <span>AI 设置</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup className='min-h-0 flex-1 px-2 py-0'>
          <SidebarGroupLabel className='flex items-center justify-between px-2'>
            最近对话
            <BookOpen className='size-3.5' />
          </SidebarGroupLabel>
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
                        isActive={view === 'chat' && activeSessionId === session.id}
                        onClick={() => onSelectSession(session.id)}
                        className='h-auto min-h-11 items-start py-2'
                      >
                        <span className='line-clamp-2 leading-5'>{session.title}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))
                )}
              </SidebarMenu>
            </ScrollArea>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className='px-3 pb-3'>
        <div className='flex min-h-11 flex-1 items-center gap-3 rounded-lg px-2'>
          <Avatar className='size-8'>
            <AvatarFallback>管</AvatarFallback>
          </Avatar>
          <div className='min-w-0 flex-1'>
            <p className='truncate text-sm font-medium'>管理员</p>
            <p className='truncate text-xs text-muted-foreground'>Pixoma 工作区</p>
          </div>
        </div>
      </SidebarFooter>
    </aside>
  )
}

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
import { cn } from '@/lib/utils'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'

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
    <aside className='flex h-full w-64 shrink-0 flex-col border-r bg-sidebar text-sidebar-foreground'>
      <div className='flex h-16 items-center gap-2 px-3'>
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

      <div className='px-3 pb-3'>
        <Button
          className='h-10 w-full justify-start'
          onClick={onNewSession}
          disabled={creating}
        >
          <MessageSquarePlus />
          {creating ? '正在创建…' : '新对话'}
        </Button>
      </div>

      <nav className='space-y-1 px-3' aria-label='Studio 功能'>
        <Button
          variant={view === 'library' ? 'secondary' : 'ghost'}
          className='h-10 w-full justify-start'
          onClick={() => onViewChange('library')}
        >
          <Library />
          资产库
        </Button>
        <Button
          variant={view === 'settings' ? 'secondary' : 'ghost'}
          className='h-10 w-full justify-start'
          onClick={() => onViewChange('settings')}
        >
          <Settings2 />
          AI 设置
        </Button>
      </nav>

      <Separator className='my-3' />
      <div className='flex items-center justify-between px-4 pb-2'>
        <span className='text-xs font-medium text-muted-foreground'>最近对话</span>
        <BookOpen className='size-3.5 text-muted-foreground' />
      </div>
      <ScrollArea className='min-h-0 flex-1 px-2'>
        <div className='space-y-1 pb-4'>
          {sessions.length === 0 ? (
            <div className='mx-2 rounded-lg border border-dashed p-4 text-center text-xs text-muted-foreground'>
              开始一次对话后，会自动保存在这里
            </div>
          ) : (
            sessions.map((session) => (
              <button
                key={session.id}
                type='button'
                onClick={() => onSelectSession(session.id)}
                className={cn(
                  'flex min-h-11 w-full items-center rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-sidebar-accent',
                  view === 'chat' && activeSessionId === session.id
                    ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                    : 'text-sidebar-foreground'
                )}
              >
                <span className='line-clamp-2 leading-5'>{session.title}</span>
              </button>
            ))
          )}
        </div>
      </ScrollArea>

      <div className='border-t p-3'>
        <div className='flex min-h-11 items-center gap-3 rounded-lg px-2'>
          <Avatar className='size-8'>
            <AvatarFallback>管</AvatarFallback>
          </Avatar>
          <div className='min-w-0 flex-1'>
            <p className='truncate text-sm font-medium'>管理员</p>
            <p className='truncate text-xs text-muted-foreground'>Pixoma 工作区</p>
          </div>
        </div>
      </div>
    </aside>
  )
}

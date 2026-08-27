import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ArrowRightFromLine } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { fetchMediaBlob, resolveMediaKey } from '@/lib/api/media'
import {
  fetchCurrentUser,
  logoutAdmin,
  type CurrentUser,
} from '@/lib/api/setup'
import { getDisplayNameInitials } from '@/lib/utils'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { Skeleton } from '@/components/ui/skeleton'
import { SignOutDialog } from '@/components/sign-out-dialog'

/** 头像存的是带鉴权的媒体 key，需拉 Blob 转 object URL 后才可作 <img> 源。 */
function useAvatarSrc(avatarUrl?: string): string | undefined {
  const key = avatarUrl ? resolveMediaKey(avatarUrl) : null
  const [loaded, setLoaded] = useState<
    { key: string; url: string } | undefined
  >()
  useEffect(() => {
    if (!key) return
    let cancelled = false
    let objectUrl: string | undefined
    fetchMediaBlob(key)
      .then((blob) => {
        if (cancelled) return
        objectUrl = URL.createObjectURL(blob)
        setLoaded({ key, url: objectUrl })
      })
      .catch(() => {
        // 加载失败时回退到首字母占位
      })
    return () => {
      cancelled = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [key])
  return loaded && loaded.key === key ? loaded.url : undefined
}

function UserIdentity({
  user,
  isLoading,
  className,
}: {
  user?: CurrentUser
  isLoading: boolean
  className?: string
}) {
  const displayName = user?.nickname || user?.username

  if (isLoading) {
    return (
      <div className={className ?? ''}>
        <Skeleton className='h-4 w-24' />
      </div>
    )
  }

  return (
    <div className={`flex min-w-0 flex-col justify-center ${className ?? ''}`}>
      <span className='truncate text-xs leading-tight text-foreground'>
        {displayName}
      </span>
    </div>
  )
}

function UserAvatar({ user }: { user?: CurrentUser }) {
  const displayName = user?.nickname || user?.username || ''
  const initials = user ? getDisplayNameInitials(user.username) : '?'
  const avatarUrl = useAvatarSrc(user?.avatar_url)
  return (
    <Avatar className='size-6 shrink-0 rounded-full'>
      {avatarUrl ? <AvatarImage src={avatarUrl} alt={displayName} /> : null}
      <AvatarFallback>{initials}</AvatarFallback>
    </Avatar>
  )
}

export function NavUser() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { state, isMobile } = useSidebar()
  const isCollapsed = state === 'collapsed' && !isMobile
  const [signOutOpen, setSignOutOpen] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const hoverTimer = useRef<number | null>(null)
  const { data: user, isLoading } = useQuery({
    queryKey: ['current-user'],
    queryFn: fetchCurrentUser,
  })

  const openMenu = () => {
    if (hoverTimer.current !== null) window.clearTimeout(hoverTimer.current)
    hoverTimer.current = null
    setMenuOpen(true)
  }

  const closeMenuSoon = () => {
    if (hoverTimer.current !== null) window.clearTimeout(hoverTimer.current)
    hoverTimer.current = window.setTimeout(() => setMenuOpen(false), 150)
  }

  async function handleSignOut() {
    await logoutAdmin()
    await navigate({ to: '/login' })
  }

  const signOutItem = (
    <DropdownMenuItem
      onClick={() => setSignOutOpen(true)}
      className='gap-2'
      data-testid='nav-user-sign-out-item'
    >
      <ArrowRightFromLine className='size-4' />
      <span>{t('common.signOut')}</span>
    </DropdownMenuItem>
  )

  if (!isCollapsed) {
    return (
      <div className='flex h-12 items-center gap-2 px-2' data-testid='nav-user'>
        <UserAvatar user={user} />
        <UserIdentity user={user} isLoading={isLoading} className='flex-1' />
        <Button
          variant='ghost'
          size='icon'
          className='shrink-0 text-muted-foreground'
          aria-label={t('common.signOut')}
          onClick={() => setSignOutOpen(true)}
          data-testid='nav-user-sign-out'
        >
          <ArrowRightFromLine className='size-4' />
        </Button>
        <SignOutDialog
          open={signOutOpen}
          onOpenChange={setSignOutOpen}
          onSignOut={handleSignOut}
        />
      </div>
    )
  }

  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
            <DropdownMenuTrigger
              asChild
              onMouseEnter={openMenu}
              onMouseLeave={closeMenuSoon}
            >
              <SidebarMenuButton
                size='lg'
                className='data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
                data-testid='nav-user-collapsed-trigger'
              >
                <UserAvatar user={user} />
              </SidebarMenuButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className='w-(--radix-dropdown-menu-trigger-width) min-w-64 rounded-lg'
              side='right'
              align='start'
              sideOffset={4}
              onMouseEnter={openMenu}
              onMouseLeave={closeMenuSoon}
            >
              <DropdownMenuLabel className='p-0 font-normal'>
                <div className='flex items-center gap-2 px-1 py-3'>
                  <UserAvatar user={user} />
                  <UserIdentity user={user} isLoading={isLoading} />
                </div>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              {signOutItem}
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
      <SignOutDialog
        open={signOutOpen}
        onOpenChange={setSignOutOpen}
        onSignOut={handleSignOut}
      />
    </>
  )
}

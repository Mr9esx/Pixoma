import { useEffect, useReducer, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { CN, US } from 'country-flag-icons/react/3x2'
import { ArrowRightFromLine, Check, Moon, Sun } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { fetchMediaBlob, resolveMediaKey } from '@/lib/api/media'
import {
  fetchCurrentUser,
  logoutAdmin,
  type CurrentUser,
} from '@/lib/api/setup'
import { setStoredLocale, type AppLocale } from '@/lib/i18n'
import { cn, getDisplayNameInitials } from '@/lib/utils'
import { useTheme } from '@/context/theme-provider'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useSidebar } from '@/components/ui/sidebar'
import { Skeleton } from '@/components/ui/skeleton'
import { ThemeSwitcher } from '@/components/kibo-ui/theme-switcher'
import { SignOutDialog } from '@/components/sign-out-dialog'

/** 头像存的是带鉴权的媒体 key，需拉 Blob 转 object URL 后才可作 <img> 源。 */
// object URL 在模块级按 key 缓存：折叠/展开会重建 NavUser 子树，
// 若每次重新请求会导致头像闪回首字母占位，这里改为复用已加载的 URL。
const avatarBlobUrlCache = new Map<string, string>()
const avatarBlobUrlLoaders = new Map<string, Promise<string>>()

function useAvatarSrc(avatarUrl?: string): string | undefined {
  const key = avatarUrl ? resolveMediaKey(avatarUrl) : null
  const [, bump] = useReducer((n: number) => n + 1, 0)

  useEffect(() => {
    if (!key) return
    if (avatarBlobUrlCache.has(key)) return
    let loader = avatarBlobUrlLoaders.get(key)
    if (!loader) {
      loader = fetchMediaBlob(key)
        .then((blob) => {
          const objectUrl = URL.createObjectURL(blob)
          avatarBlobUrlCache.set(key, objectUrl)
          // 头像更新后释放旧的 object URL
          for (const [oldKey, oldUrl] of avatarBlobUrlCache) {
            if (oldKey !== key) {
              avatarBlobUrlCache.delete(oldKey)
              URL.revokeObjectURL(oldUrl)
            }
          }
          return objectUrl
        })
        .finally(() => {
          avatarBlobUrlLoaders.delete(key)
        })
      avatarBlobUrlLoaders.set(key, loader)
    }
    let cancelled = false
    loader.then(
      () => {
        if (!cancelled) bump()
      },
      () => {
        // 加载失败时回退到首字母占位
      }
    )
    return () => {
      cancelled = true
    }
  }, [key])

  return key ? avatarBlobUrlCache.get(key) : undefined
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
      <span className='truncate text-sm leading-tight text-foreground'>
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
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { state, isMobile } = useSidebar()
  const { theme, setTheme } = useTheme()
  const isCollapsed = state === 'collapsed' && !isMobile
  const [signOutOpen, setSignOutOpen] = useState(false)
  const { data: user, isLoading } = useQuery({
    queryKey: ['current-user'],
    queryFn: fetchCurrentUser,
  })

  const currentLocale: AppLocale = i18n.language.startsWith('en') ? 'en' : 'zh'

  // 主题切换时同步 <meta name="theme-color">，与 sidecar 页面底色保持一致
  useEffect(() => {
    const themeColor = theme === 'dark' ? '#020817' : '#fff'
    const meta = document.querySelector("meta[name='theme-color']")
    if (meta) meta.setAttribute('content', themeColor)
  }, [theme])

  function switchLocale(locale: AppLocale) {
    void i18n.changeLanguage(locale)
    setStoredLocale(locale)
  }

  async function handleSignOut() {
    await logoutAdmin()
    await navigate({ to: '/login' })
  }

  const menuContent = (
    <>
      <DropdownMenuLabel className='p-0 font-normal'>
        <div className='flex items-center gap-2 px-1 py-3'>
          <UserAvatar user={user} />
          <UserIdentity user={user} isLoading={isLoading} />
        </div>
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuLabel className='px-2 text-xs font-medium text-muted-foreground'>
        {t('lang.switch')}
      </DropdownMenuLabel>
      <DropdownMenuGroup>
        <DropdownMenuItem
          onClick={() => switchLocale('zh')}
          className='gap-2'
          data-testid='nav-user-lang-zh'
        >
          <CN className='size-4' />
          <span>{t('lang.zh')}</span>
          <Check
            size={14}
            className={cn('ms-auto', currentLocale !== 'zh' && 'hidden')}
          />
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => switchLocale('en')}
          className='gap-2'
          data-testid='nav-user-lang-en'
        >
          <US className='size-4' />
          <span>{t('lang.en')}</span>
          <Check
            size={14}
            className={cn('ms-auto', currentLocale !== 'en' && 'hidden')}
          />
        </DropdownMenuItem>
      </DropdownMenuGroup>
      {isCollapsed ? (
        <>
          <DropdownMenuSeparator />
          <DropdownMenuLabel className='px-2 text-xs font-medium text-muted-foreground'>
            {t('common.commandTheme')}
          </DropdownMenuLabel>
          <DropdownMenuItem
            onClick={() => setTheme('light')}
            className='gap-2'
            data-testid='nav-user-theme-light'
          >
            <Sun className='size-4' />
            <span>{t('theme.light')}</span>
            <Check
              size={14}
              className={cn('ms-auto', theme !== 'light' && 'hidden')}
            />
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => setTheme('dark')}
            className='gap-2'
            data-testid='nav-user-theme-dark'
          >
            <Moon className='size-4' />
            <span>{t('theme.dark')}</span>
            <Check
              size={14}
              className={cn('ms-auto', theme !== 'dark' && 'hidden')}
            />
          </DropdownMenuItem>
          <DropdownMenuSeparator />
        </>
      ) : null}
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem
          onClick={() => setSignOutOpen(true)}
          className='gap-2'
          data-testid='nav-user-sign-out-item'
        >
          <ArrowRightFromLine className='size-4' />
          <span>{t('common.signOut')}</span>
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </>
  )

  return (
    <>
      <div
        className='flex h-12 w-full items-center gap-2 pe-0'
        data-testid='nav-user'
      >
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant='ghost'
              size='sm'
              className='flex h-9 min-w-0 flex-1 cursor-pointer items-center gap-2 self-center rounded-lg px-2 text-left outline-none hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-ring data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
              data-testid='nav-user-menu-trigger'
            >
              <UserAvatar user={user} />
              <UserIdentity
                user={user}
                isLoading={isLoading}
                className={cn('flex-1', isCollapsed && 'hidden')}
              />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className='min-w-48 rounded-lg'
            align='start'
            side={isCollapsed ? 'right' : 'top'}
            sideOffset={4}
            onCloseAutoFocus={(event) => event.preventDefault()}
          >
            {menuContent}
          </DropdownMenuContent>
        </DropdownMenu>

        <div
          className={cn(
            'shrink-0 transition-opacity duration-300 ease-in-out',
            isCollapsed ? 'pointer-events-none opacity-0' : 'opacity-100'
          )}
        >
          <ThemeSwitcher value={theme} onChange={setTheme} className='my-0.5' />
        </div>
      </div>

      <SignOutDialog
        open={signOutOpen}
        onOpenChange={setSignOutOpen}
        onSignOut={handleSignOut}
      />
    </>
  )
}

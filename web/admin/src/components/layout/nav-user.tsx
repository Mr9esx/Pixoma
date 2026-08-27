import { useEffect, useReducer, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  ArrowRightFromLine,
  Check,
  Globe,
  Monitor,
  Moon,
  MoreHorizontal,
  Sun,
} from 'lucide-react'
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
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  useSidebar,
} from '@/components/ui/sidebar'
import { Skeleton } from '@/components/ui/skeleton'
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

function MenuButton({
  onClick,
  active = false,
  destructive = false,
  testId,
  children,
}: {
  onClick: () => void
  active?: boolean
  destructive?: boolean
  testId?: string
  children: ReactNode
}) {
  return (
    <button
      type='button'
      data-testid={testId}
      onClick={onClick}
      className={cn(
        'relative flex w-full cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-start text-sm outline-none hover:bg-accent hover:text-accent-foreground',
        destructive && 'text-destructive hover:text-destructive'
      )}
    >
      {children}
      {active ? <Check size={14} className='ms-auto' /> : null}
    </button>
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
      {isCollapsed ? (
        <>
          <DropdownMenuLabel className='p-0 font-normal'>
            <div className='flex items-center gap-2 px-1 py-3'>
              <UserAvatar user={user} />
              <UserIdentity user={user} isLoading={isLoading} />
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
        </>
      ) : null}

      <DropdownMenuLabel className='px-2 text-xs font-medium text-muted-foreground'>
        {t('lang.switch')}
      </DropdownMenuLabel>
      <DropdownMenuItem
        onClick={() => switchLocale('zh')}
        className='gap-2'
        data-testid='nav-user-lang-zh'
      >
        <Globe className='size-4' />
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
        <Globe className='size-4' />
        <span>{t('lang.en')}</span>
        <Check
          size={14}
          className={cn('ms-auto', currentLocale !== 'en' && 'hidden')}
        />
      </DropdownMenuItem>

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
      <DropdownMenuItem
        onClick={() => setTheme('system')}
        className='gap-2'
        data-testid='nav-user-theme-system'
      >
        <Monitor className='size-4' />
        <span>{t('theme.system')}</span>
        <Check
          size={14}
          className={cn('ms-auto', theme !== 'system' && 'hidden')}
        />
      </DropdownMenuItem>

      <DropdownMenuSeparator />

      <DropdownMenuItem
        onClick={() => setSignOutOpen(true)}
        className='gap-2 text-destructive focus:text-destructive'
        data-testid='nav-user-sign-out-item'
      >
        <ArrowRightFromLine className='size-4' />
        <span>{t('common.signOut')}</span>
      </DropdownMenuItem>
    </>
  )

  // 折叠态菜单：纯 CSS hover 弹出（group + group-hover），不依赖任何 JS 开关状态
  const collapsedMenuContent = (
    <div className='w-64 p-1' data-testid='nav-user-collapsed-menu'>
      <div className='flex items-center gap-2 px-1 py-3'>
        <UserAvatar user={user} />
        <UserIdentity user={user} isLoading={isLoading} />
      </div>
      <div className='-mx-1 my-1 h-px bg-border' />

      <div className='px-2 py-1.5 text-xs font-medium text-muted-foreground'>
        {t('lang.switch')}
      </div>
      <MenuButton
        onClick={() => switchLocale('zh')}
        active={currentLocale === 'zh'}
        testId='nav-user-lang-zh'
      >
        <Globe className='size-4' />
        <span>{t('lang.zh')}</span>
      </MenuButton>
      <MenuButton
        onClick={() => switchLocale('en')}
        active={currentLocale === 'en'}
        testId='nav-user-lang-en'
      >
        <Globe className='size-4' />
        <span>{t('lang.en')}</span>
      </MenuButton>

      <div className='-mx-1 my-1 h-px bg-border' />
      <div className='px-2 py-1.5 text-xs font-medium text-muted-foreground'>
        {t('common.commandTheme')}
      </div>
      <MenuButton
        onClick={() => setTheme('light')}
        active={theme === 'light'}
        testId='nav-user-theme-light'
      >
        <Sun className='size-4' />
        <span>{t('theme.light')}</span>
      </MenuButton>
      <MenuButton
        onClick={() => setTheme('dark')}
        active={theme === 'dark'}
        testId='nav-user-theme-dark'
      >
        <Moon className='size-4' />
        <span>{t('theme.dark')}</span>
      </MenuButton>
      <MenuButton
        onClick={() => setTheme('system')}
        active={theme === 'system'}
        testId='nav-user-theme-system'
      >
        <Monitor className='size-4' />
        <span>{t('theme.system')}</span>
      </MenuButton>

      <div className='-mx-1 my-1 h-px bg-border' />
      <MenuButton
        onClick={() => setSignOutOpen(true)}
        destructive
        testId='nav-user-sign-out-item'
      >
        <ArrowRightFromLine className='size-4' />
        <span>{t('common.signOut')}</span>
      </MenuButton>
    </div>
  )

  return (
    <>
      {isCollapsed ? (
        <div
          className='flex h-12 items-center justify-center'
          data-testid='nav-user-collapsed'
        >
          <div className='relative w-fit'>
            <button
              type='button'
              aria-label='user menu'
              className='nav-user-collapsed-trigger flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-md outline-none hover:bg-sidebar-accent'
              data-testid='nav-user-collapsed-trigger'
            >
              <UserAvatar user={user} />
            </button>
            <div className='nav-user-menu absolute left-full bottom-0 z-50 max-h-[calc(100dvh-5rem)] min-w-64 overflow-y-auto rounded-lg border bg-popover text-popover-foreground shadow-md'>
              {collapsedMenuContent}
            </div>
          </div>
        </div>
      ) : (
        <div
          className='flex h-12 items-center gap-2 px-2'
          data-testid='nav-user'
        >
          <UserAvatar user={user} />
          <UserIdentity
            user={user}
            isLoading={isLoading}
            className='flex-1'
          />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant='ghost'
                size='icon'
                className='shrink-0 text-muted-foreground data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
                aria-label={t('common.moreActions')}
                data-testid='nav-user-menu-trigger'
              >
                <MoreHorizontal className='size-4' />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className='min-w-48 rounded-lg'
              align='end'
              side='top'
              sideOffset={4}
              onCloseAutoFocus={(event) => event.preventDefault()}
            >
              {menuContent}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}

      <SignOutDialog
        open={signOutOpen}
        onOpenChange={setSignOutOpen}
        onSignOut={handleSignOut}
      />
    </>
  )
}

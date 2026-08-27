import { Link, useRouterState } from '@tanstack/react-router'
import { MENU_GROUPS } from '@/config/menu'
import { Menu, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useLayout } from '@/context/layout-provider'
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
  SidebarRail,
  useSidebar,
} from '@/components/ui/sidebar'
import { ThemeSwitch } from '@/components/theme-switch'
import { AppTitle } from './app-title'
import { LanguageSwitcher } from './language-switcher'
import { NavUser } from './nav-user'

export function AppSidebar() {
  const { t } = useTranslation()
  const { collapsible, variant } = useLayout()
  const { setOpenMobile, state, toggleSidebar, isMobile } = useSidebar()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const isCollapsed = state === 'collapsed'

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader>
        <AppTitle />
      </SidebarHeader>
      <SidebarContent className='gap-1'>
        {isCollapsed ? (
          <SidebarMenu className='px-2'>
            <SidebarMenuItem>
              <SidebarMenuButton
                onClick={toggleSidebar}
                tooltip={t('common.expandMenu')}
              >
                {isMobile ? <X /> : <Menu />}
                <span className='sr-only'>Toggle Sidebar</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        ) : null}
        {MENU_GROUPS.map((group) => (
          <SidebarGroup key={group.id} className='px-2 py-1'>
            {group.titleKey ? (
              <SidebarGroupLabel>{t(group.titleKey)}</SidebarGroupLabel>
            ) : null}
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => {
                  const label = t(item.titleKey)
                  const isActive =
                    item.path === '/'
                      ? pathname === '/'
                      : pathname === item.path ||
                        pathname.startsWith(`${item.path}/`)

                  return (
                    <SidebarMenuItem key={item.id}>
                      <SidebarMenuButton
                        asChild
                        isActive={isActive}
                        tooltip={label}
                      >
                        <Link
                          to={item.path}
                          onClick={() => setOpenMobile(false)}
                        >
                          <item.icon />
                          <span>{label}</span>
                        </Link>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}

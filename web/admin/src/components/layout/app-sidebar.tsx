import { useQuery } from '@tanstack/react-query'
import { Link, useRouterState } from '@tanstack/react-router'
import { MENU_GROUPS, filterMenuGroupsForDemo } from '@/config/menu'
import { Menu } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { fetchSetupStatus } from '@/lib/api/setup'
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
  useSidebar,
} from '@/components/ui/sidebar'
import { AppTitle } from './app-title'
import { NavUser } from './nav-user'

export function AppSidebar() {
  const { t } = useTranslation()
  const { collapsible, variant } = useLayout()
  const { setOpenMobile, state, toggleSidebar, isMobile } = useSidebar()
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  const isCollapsed = !isMobile && state === 'collapsed'
  const { data: setupStatus } = useQuery({
    queryKey: ['setup-status'],
    queryFn: fetchSetupStatus,
  })
  const isLiveDemo = setupStatus?.live_demo === true
  const menuGroups = filterMenuGroupsForDemo(MENU_GROUPS, isLiveDemo)

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
                <Menu />
                <span className='sr-only'>Toggle Sidebar</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        ) : null}
        {menuGroups.map((group) => (
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
    </Sidebar>
  )
}

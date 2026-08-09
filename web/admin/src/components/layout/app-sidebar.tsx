import { Link, useRouterState } from '@tanstack/react-router'
import { MENU_ITEMS } from '@/config/menu'
import { useLayout } from '@/context/layout-provider'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from '@/components/ui/sidebar'
import { AppTitle } from './app-title'

/** Temporary visible labels until Task 5 wires i18n via titleKey. */
const TEMP_LABELS: Record<string, string> = {
  dashboard: 'Dashboard',
  instances: 'Instances',
  cases: 'Cases',
  tasks: 'Tasks',
  users: 'Users',
  sessions: 'Sessions',
}

export function AppSidebar() {
  const { collapsible, variant } = useLayout()
  const { setOpenMobile } = useSidebar()
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader>
        <AppTitle />
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {MENU_ITEMS.map((item) => {
                const label = TEMP_LABELS[item.id] ?? item.id
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
                      <Link to={item.path} onClick={() => setOpenMobile(false)}>
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
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}

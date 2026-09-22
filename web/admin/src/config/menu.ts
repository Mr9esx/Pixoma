import {
  LayoutDashboard,
  Server,
  Boxes,
  ListTodo,
  Users,
  MessagesSquare,
  Radio,
  Settings,
  Waypoints,
  Zap,
  WandSparkles,
  type LucideIcon,
} from 'lucide-react'

type MenuItem = {
  id: string
  titleKey: string
  path: string
  icon: LucideIcon
}

type MenuGroup = {
  id: string
  titleKey?: string
  items: readonly MenuItem[]
}

const ITEMS: readonly MenuItem[] = [
  {
    id: 'dashboard',
    titleKey: 'menu.dashboard',
    path: '/',
    icon: LayoutDashboard,
  },
  {
    id: 'quick-config',
    titleKey: 'menu.quickConfig',
    path: '/quick-config',
    icon: Zap,
  },
  {
    id: 'studio',
    titleKey: 'menu.studio',
    path: '/studio',
    icon: WandSparkles,
  },
  { id: 'cases', titleKey: 'menu.cases', path: '/cases', icon: Boxes },
  { id: 'channels', titleKey: 'menu.channels', path: '/channels', icon: Radio },
  { id: 'topics', titleKey: 'menu.topics', path: '/topics', icon: Waypoints },
  { id: 'edges', titleKey: 'menu.edges', path: '/edges', icon: Server },
  { id: 'tasks', titleKey: 'menu.tasks', path: '/tasks', icon: ListTodo },
  {
    id: 'sessions',
    titleKey: 'menu.sessions',
    path: '/sessions',
    icon: MessagesSquare,
  },
  { id: 'users', titleKey: 'menu.users', path: '/users', icon: Users },
  {
    id: 'settings',
    titleKey: 'menu.settings',
    path: '/settings',
    icon: Settings,
  },
] as const

/** 按 id 取出菜单项。分组只声明顺序，规则写在 MENU_GROUPS 里。 */
function item(id: string): MenuItem {
  const found = ITEMS.find((entry) => entry.id === id)
  if (!found) throw new Error(`菜单项 ${id} 未定义`)
  return found
}

export const MENU_GROUPS: readonly MenuGroup[] = [
  {
    id: 'overview',
    items: [item('dashboard'), item('quick-config'), item('studio')],
  },
  {
    id: 'config',
    titleKey: 'menu.groupConfig',
    items: [item('cases'), item('channels'), item('topics'), item('edges')],
  },
  {
    id: 'operations',
    titleKey: 'menu.groupOperations',
    items: [item('tasks'), item('sessions'), item('users')],
  },
  {
    id: 'system',
    titleKey: 'menu.groupSystem',
    items: [item('settings')],
  },
] as const

export function filterMenuGroupsForDemo(
  groups: readonly MenuGroup[],
  isLiveDemo: boolean
): readonly MenuGroup[] {
  if (!isLiveDemo) return groups
  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => item.id !== 'settings'),
    }))
    .filter((group) => group.items.length > 0)
}

export const MENU_ITEMS: readonly MenuItem[] = ITEMS

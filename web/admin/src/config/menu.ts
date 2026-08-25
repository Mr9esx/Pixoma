import {
  LayoutDashboard,
  Server,
  Boxes,
  ListTodo,
  Users,
  MessagesSquare,
  Radio,
  Settings,
  Tags,
  Zap,
  type LucideIcon,
} from 'lucide-react'

export type MenuItem = {
  id: string
  titleKey: string
  path: string
  icon: LucideIcon
}

export type MenuGroup = {
  id: string
  titleKey?: string
  items: readonly MenuItem[]
}

const ITEMS: readonly MenuItem[] = [
  { id: 'dashboard', titleKey: 'menu.dashboard', path: '/', icon: LayoutDashboard },
  { id: 'quick-config', titleKey: 'menu.quickConfig', path: '/quick-config', icon: Zap },
  { id: 'cases', titleKey: 'menu.cases', path: '/cases', icon: Boxes },
  { id: 'channels', titleKey: 'menu.channels', path: '/channels', icon: Radio },
  { id: 'topics', titleKey: 'menu.topics', path: '/topics', icon: Tags },
  { id: 'edges', titleKey: 'menu.edges', path: '/edges', icon: Server },
  { id: 'tasks', titleKey: 'menu.tasks', path: '/tasks', icon: ListTodo },
  { id: 'sessions', titleKey: 'menu.sessions', path: '/sessions', icon: MessagesSquare },
  { id: 'users', titleKey: 'menu.users', path: '/users', icon: Users },
  { id: 'settings', titleKey: 'menu.settings', path: '/settings', icon: Settings },
] as const

export const MENU_GROUPS: readonly MenuGroup[] = [
  { id: 'overview', items: [ITEMS[0], ITEMS[1]] },
  {
    id: 'config',
    titleKey: 'menu.groupConfig',
    items: [ITEMS[2], ITEMS[3], ITEMS[4], ITEMS[5]],
  },
  {
    id: 'operations',
    titleKey: 'menu.groupOperations',
    items: [ITEMS[6], ITEMS[7], ITEMS[8]],
  },
  {
    id: 'system',
    titleKey: 'menu.groupSystem',
    items: [ITEMS[9]],
  },
] as const

export const MENU_ITEMS: readonly MenuItem[] = ITEMS

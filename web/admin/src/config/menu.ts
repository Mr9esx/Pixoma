import {
  LayoutDashboard,
  Server,
  Boxes,
  ListTodo,
  Users,
  MessagesSquare,
  Radio,
  Settings,
  type LucideIcon,
} from 'lucide-react'

export type MenuItem = {
  id: string
  titleKey: string
  path: string
  icon: LucideIcon
}

export const MENU_ITEMS: readonly MenuItem[] = [
  { id: 'dashboard', titleKey: 'menu.dashboard', path: '/', icon: LayoutDashboard },
  { id: 'edges', titleKey: 'menu.edges', path: '/edges', icon: Server },
  { id: 'cases', titleKey: 'menu.cases', path: '/cases', icon: Boxes },
  { id: 'channels', titleKey: 'menu.channels', path: '/channels', icon: Radio },
  { id: 'tasks', titleKey: 'menu.tasks', path: '/tasks', icon: ListTodo },
  { id: 'users', titleKey: 'menu.users', path: '/users', icon: Users },
  { id: 'sessions', titleKey: 'menu.sessions', path: '/sessions', icon: MessagesSquare },
  { id: 'settings', titleKey: 'menu.settings', path: '/settings', icon: Settings },
] as const

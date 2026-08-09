import {
  LayoutDashboard,
  Server,
  Boxes,
  ListTodo,
  Users,
  MessagesSquare,
  Keyboard,
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
  { id: 'instances', titleKey: 'menu.instances', path: '/instances', icon: Server },
  { id: 'cases', titleKey: 'menu.cases', path: '/cases', icon: Boxes },
  { id: 'tg-menu', titleKey: 'menu.tgMenu', path: '/tg-menu', icon: Keyboard },
  { id: 'tasks', titleKey: 'menu.tasks', path: '/tasks', icon: ListTodo },
  { id: 'users', titleKey: 'menu.users', path: '/users', icon: Users },
  { id: 'sessions', titleKey: 'menu.sessions', path: '/sessions', icon: MessagesSquare },
] as const

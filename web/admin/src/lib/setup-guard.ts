import type { SetupStatus } from '@/lib/api/setup'

export function nextAdminPath(
  status: SetupStatus,
  current: string,
): '/setup' | '/login' | '/' | null {
  if (status.restart_required) {
    return current === '/setup' ? null : '/setup'
  }
  if (!status.initialized) {
    if (!status.authenticated) {
      return current === '/login' ? null : '/login'
    }
    return current === '/setup' ? null : '/setup'
  }
  if (!status.authenticated) {
    return current === '/login' ? null : '/login'
  }
  if (current === '/login' || current === '/setup') {
    return '/'
  }
  return null
}

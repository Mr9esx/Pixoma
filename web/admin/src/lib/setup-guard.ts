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


// nextRegistrationPath decides where an unauthenticated visitor on the
// self-registration page should go. The page is only reachable on an
// initialized platform with open registration switched on.
export function nextRegistrationPath(
  status: SetupStatus,
  open: boolean,
  current: string,
): '/setup' | '/login' | '/' | null {
  if (status.restart_required) {
    return '/setup'
  }
  if (!status.initialized) {
    return status.authenticated ? '/setup' : '/login'
  }
  if (status.authenticated) {
    return '/'
  }
  if (!open) {
    return current === '/login' ? null : '/login'
  }
  return null
}

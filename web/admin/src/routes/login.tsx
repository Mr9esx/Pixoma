import { createFileRoute, redirect } from '@tanstack/react-router'
import { fetchSetupStatus } from '@/lib/api/setup'
import { nextAdminPath } from '@/lib/setup-guard'
import { LoginPage } from '@/features/setup/login-page'

export const Route = createFileRoute('/login')({
  beforeLoad: async () => {
    const status = await fetchSetupStatus()
    const next = nextAdminPath(status, '/login')
    if (next) {
      throw redirect({ to: next })
    }
    return { status }
  },
  component: LoginRouteComponent,
})

function LoginRouteComponent() {
  const { status } = Route.useRouteContext()
  return <LoginPage status={status} />
}

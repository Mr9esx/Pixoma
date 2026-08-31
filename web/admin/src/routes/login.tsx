import { createFileRoute, redirect } from '@tanstack/react-router'
import { fetchSetupStatus, fetchRegistrationStatus } from '@/lib/api/setup'
import { nextAdminPath } from '@/lib/setup-guard'
import { LoginPage } from '@/features/setup/login-page'

export const Route = createFileRoute('/login')({
  beforeLoad: async () => {
    const status = await fetchSetupStatus()
    const registrationOpen = status.live_demo
      ? false
      : await fetchRegistrationStatus()
          .then((r) => r.enabled)
          .catch(() => false)
    const next = nextAdminPath(status, '/login')
    if (next) {
      throw redirect({ to: next })
    }
    return { status, statusLiveDemo: status.live_demo, registrationOpen }
  },
  component: LoginRouteComponent,
})

function LoginRouteComponent() {
  const { status, registrationOpen } = Route.useRouteContext()
  return <LoginPage status={status} registrationOpen={registrationOpen} />
}

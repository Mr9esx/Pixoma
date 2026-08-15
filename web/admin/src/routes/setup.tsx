import { createFileRoute, redirect } from '@tanstack/react-router'
import { SetupWizard } from '@/features/setup/setup-wizard'
import { fetchSetupStatus } from '@/lib/api/setup'
import { nextAdminPath } from '@/lib/setup-guard'

export const Route = createFileRoute('/setup')({
  beforeLoad: async () => {
    const status = await fetchSetupStatus()
    const next = nextAdminPath(status, '/setup')
    if (next) {
      throw redirect({ to: next })
    }
    return { status }
  },
  component: SetupPage,
})

function SetupPage() {
  const { status } = Route.useRouteContext()
  return <SetupWizard status={status} />
}

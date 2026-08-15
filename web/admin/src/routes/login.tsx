import { createFileRoute, redirect } from '@tanstack/react-router'
import { LoginPage } from '@/features/setup/login-page'
import { fetchSetupStatus } from '@/lib/api/setup'
import { nextAdminPath } from '@/lib/setup-guard'

export const Route = createFileRoute('/login')({
  beforeLoad: async () => {
    const status = await fetchSetupStatus()
    const next = nextAdminPath(status, '/login')
    if (next) {
      throw redirect({ to: next })
    }
  },
  component: LoginPage,
})

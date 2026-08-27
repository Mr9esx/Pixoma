import { createFileRoute, redirect } from '@tanstack/react-router'
import { fetchSetupStatus, fetchRegistrationStatus } from '@/lib/api/setup'
import { nextRegistrationPath } from '@/lib/setup-guard'
import { RegisterForm } from '@/features/auth/register-form'

export const Route = createFileRoute('/register')({
  beforeLoad: async () => {
    const status = await fetchSetupStatus()
    const reg = await fetchRegistrationStatus()
    const next = nextRegistrationPath(status, reg.enabled, '/register')
    if (next) {
      throw redirect({ to: next })
    }
    return { status }
  },
  component: RegisterRouteComponent,
})

function RegisterRouteComponent() {
  return <RegisterForm />
}

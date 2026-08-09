import { createFileRoute } from '@tanstack/react-router'
import { DashboardPage } from '@/features/dashboard/dashboard-page'

export const Route = createFileRoute('/_app/')({
  component: DashboardPage,
})

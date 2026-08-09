import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/')({
  component: () => <div data-testid='dashboard-page'>Dashboard</div>,
})

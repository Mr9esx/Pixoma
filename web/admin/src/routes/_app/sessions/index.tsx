import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/sessions/')({
  component: () => <div data-testid='sessions-page'>Sessions</div>,
})

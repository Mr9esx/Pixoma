import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/tasks/')({
  component: () => <div data-testid='tasks-page'>Tasks</div>,
})

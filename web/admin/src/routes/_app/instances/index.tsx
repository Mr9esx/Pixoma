import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/instances/')({
  component: () => <div data-testid='instances-page'>Instances</div>,
})

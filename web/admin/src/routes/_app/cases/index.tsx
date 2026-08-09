import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/cases/')({
  component: () => <div data-testid='cases-page'>Cases</div>,
})

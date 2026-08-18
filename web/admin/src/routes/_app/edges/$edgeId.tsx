import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/edges/$edgeId')({
  component: () => null,
})

import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/users/$userId')({
  component: () => null,
})

import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/sessions/$sessionId')({
  component: () => null,
})

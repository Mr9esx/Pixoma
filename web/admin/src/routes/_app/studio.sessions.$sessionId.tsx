import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/studio/sessions/$sessionId')({
  validateSearch: (search: Record<string, unknown>): { panel?: 'assets' } => ({
    panel: search.panel === 'assets' ? 'assets' : undefined,
  }),
})

import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/studio/sessions/$sessionId')({
  validateSearch: (search: Record<string, unknown>): { panel?: 'assets' | 'flow' | 'tasks' } => ({
    panel: search.panel === 'assets' || search.panel === 'flow' || search.panel === 'tasks' ? search.panel : undefined,
  }),
})

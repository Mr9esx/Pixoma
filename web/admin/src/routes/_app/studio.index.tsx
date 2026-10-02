import { createFileRoute, redirect } from '@tanstack/react-router'

type LegacyStudioSearch = {
  project?: string
  view?: 'library' | 'settings'
  session?: string
  section?: 'models' | 'skills' | 'mcp' | 'workflows' | 'data'
  trace?: boolean
  panel?: 'assets' | 'flow' | 'tasks'
}

export const Route = createFileRoute('/_app/studio/')({
  validateSearch: (search: Record<string, unknown>): LegacyStudioSearch => ({
    project: typeof search.project === 'string' ? search.project : undefined,
    view:
      search.view === 'library' || search.view === 'settings'
        ? search.view
        : undefined,
    session: typeof search.session === 'string' ? search.session : undefined,
    section:
      search.section === 'models' ||
      search.section === 'skills' ||
      search.section === 'mcp' ||
      search.section === 'workflows' ||
      search.section === 'data'
        ? search.section
        : undefined,
    trace: search.trace === true || search.trace === 'true' ? true : undefined,
    panel: search.panel === 'assets' || search.panel === 'flow' || search.panel === 'tasks' ? search.panel : undefined,
  }),
  beforeLoad: ({ search }) => {
    if (search.view === 'library') {
      throw redirect({ to: '/studio/library', replace: true })
    }
    if (search.view === 'settings') {
      const section =
        search.section === 'skills' ||
        search.section === 'mcp' ||
        search.section === 'workflows' ||
        search.section === 'data'
          ? search.section
          : 'models'
      throw redirect({
        to: '/studio/settings/$section',
        params: { section },
        replace: true,
      })
    }
    if (typeof search.session === 'string' && search.session.trim()) {
      if (search.trace) {
        throw redirect({
          to: '/studio/sessions/$sessionId/trace',
          params: { sessionId: search.session },
          search: {},
          replace: true,
        })
      }
      throw redirect({
        to: '/studio/sessions/$sessionId',
        params: { sessionId: search.session },
        search: search.panel ? { panel: search.panel } : {},
        replace: true,
      })
    }
  },
})

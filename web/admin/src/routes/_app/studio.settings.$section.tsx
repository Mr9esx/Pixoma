import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/_app/studio/settings/$section')({
  beforeLoad: ({ params }) => {
    if (
      params.section !== 'models' &&
      params.section !== 'skills' &&
      params.section !== 'mcp' &&
      params.section !== 'workflows'
    ) {
      throw redirect({
        to: '/studio/settings/$section',
        params: { section: 'models' },
        replace: true,
      })
    }
  },
})

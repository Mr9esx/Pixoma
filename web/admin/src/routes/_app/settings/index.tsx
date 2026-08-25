import { createFileRoute, useSearch } from '@tanstack/react-router'
import { SettingsPage } from '@/features/settings/settings-page'

export const Route = createFileRoute('/_app/settings/')({
  component: SettingsRoute,
  validateSearch: (search: Record<string, unknown>) => ({
    tab: typeof search.tab === 'string' ? search.tab : undefined,
  }),
})

function SettingsRoute() {
  const { tab } = useSearch({ from: '/_app/settings/' })
  return <SettingsPage initialTab={tab} />
}

import {
  createFileRoute,
  redirect,
  useSearch,
} from '@tanstack/react-router'
import { SettingsPage } from '@/features/settings/settings-page'
import { fetchSetupStatus } from '@/lib/api/setup'

export const Route = createFileRoute('/_app/settings/')({
  beforeLoad: async () => {
    const status = await fetchSetupStatus()
    if (status.live_demo) {
      throw redirect({ to: '/' })
    }
  },
  component: SettingsRoute,
  validateSearch: (search: Record<string, unknown>) => ({
    tab: typeof search.tab === 'string' ? search.tab : undefined,
  }),
})

function SettingsRoute() {
  const { tab } = useSearch({ from: '/_app/settings/' })
  return <SettingsPage initialTab={tab} />
}

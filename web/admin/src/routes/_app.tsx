import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { fetchSetupStatus } from '@/lib/api/setup'
import { getCookie } from '@/lib/cookies'
import { nextAdminPath } from '@/lib/setup-guard'
import { cn } from '@/lib/utils'
import { LayoutProvider } from '@/context/layout-provider'
import { SearchProvider } from '@/context/search-provider'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { AppHeader } from '@/components/layout/app-header'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { contentRegionClassName } from '@/components/layout/content-region'
import { SkipToMain } from '@/components/skip-to-main'

export const Route = createFileRoute('/_app')({
  beforeLoad: async () => {
    const status = await fetchSetupStatus()
    const next = nextAdminPath(status, '/')
    if (next) {
      throw redirect({ to: next })
    }
  },
  component: AppLayout,
})

function AppLayout() {
  const defaultOpen = getCookie('sidebar_state') !== 'false'

  return (
    <SearchProvider>
      <LayoutProvider>
        <SidebarProvider defaultOpen={defaultOpen}>
          <SkipToMain />
          <AppSidebar />
          <SidebarInset
            className={cn(
              '@container/content',
              'min-h-0 overflow-hidden',
              'h-svh',
              'md:peer-data-[variant=inset]:h-[calc(100svh-(var(--spacing)*4))]'
            )}
          >
            <AppHeader />
            <div className={cn(contentRegionClassName, 'p-4')}>
              <Outlet />
            </div>
          </SidebarInset>
        </SidebarProvider>
      </LayoutProvider>
    </SearchProvider>
  )
}

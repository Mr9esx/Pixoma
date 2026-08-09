import { createFileRoute, Outlet } from '@tanstack/react-router'
import { getCookie } from '@/lib/cookies'
import { cn } from '@/lib/utils'
import { LayoutProvider } from '@/context/layout-provider'
import { SearchProvider } from '@/context/search-provider'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { SkipToMain } from '@/components/skip-to-main'

export const Route = createFileRoute('/_app')({
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
            <div className='flex min-h-0 flex-1 flex-col overflow-hidden p-4'>
              <Outlet />
            </div>
          </SidebarInset>
        </SidebarProvider>
      </LayoutProvider>
    </SearchProvider>
  )
}

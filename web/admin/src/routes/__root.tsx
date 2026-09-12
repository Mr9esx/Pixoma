import { type QueryClient } from '@tanstack/react-query'
import { createRootRouteWithContext, Outlet } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { useTranslation } from 'react-i18next'
import { Toaster } from '@/components/ui/sonner'
import { NavigationProgress } from '@/components/navigation-progress'
import { GeneralError } from '@/features/errors/general-error'
import { NotFoundError } from '@/features/errors/not-found-error'

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  component: () => {
    return (
      <>
        <SkipToContent />
        <NavigationProgress />
        <div id='main-content'>
          <Outlet />
        </div>
        <Toaster duration={5000} />
        {import.meta.env.MODE === 'development' && (
          <div className='pointer-events-none fixed inset-0 z-2147483647'>
            <TanStackRouterDevtools position='bottom-right' />
          </div>
        )}
      </>
    )
  },
  notFoundComponent: NotFoundError,
  errorComponent: GeneralError,
})

function SkipToContent() {
  const { t } = useTranslation()
  return (
    <a
      href='#main-content'
      className='sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:rounded-md focus:bg-background focus:px-3 focus:py-2 focus:text-sm focus:ring-2 focus:ring-ring'
    >
      {t('a11y.skipToContent')}
    </a>
  )
}

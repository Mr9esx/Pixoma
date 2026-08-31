import { Logo } from '@/assets/logo'
import { SidebarTrigger } from '@/components/ui/sidebar'

export function AppHeader() {
  return (
    <header className='flex h-12 shrink-0 items-center gap-2 border-b bg-background px-3 md:hidden'>
      <SidebarTrigger className='size-8' />
      <div
        className='h-4 w-px bg-border'
        data-testid='mobile-shell-separator'
      />
      <div
        className='flex min-w-0 items-center gap-1.5 font-bold'
        data-testid='mobile-shell-title'
      >
        <Logo className='size-5 rounded-md' />
        <span className='truncate'>Pixoma</span>
      </div>
    </header>
  )
}

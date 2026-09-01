import { Menu, X } from 'lucide-react'
import { Logo } from '@/assets/logo'
import { cn } from '@/lib/utils'
import { useTheme } from '@/context/theme-provider'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { Button } from '../ui/button'

export function AppTitle() {
  const { state, isMobile } = useSidebar()
  const { theme } = useTheme()

  if (state === 'collapsed' && !isMobile) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            asChild
            tooltip='Pixoma'
            className='justify-center group-data-[collapsible=icon]:p-0!'
          >
            <span>
              <Logo className='size-5 rounded-md' />
            </span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    )
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          size='lg'
          className='gap-0 py-0 pe-0 hover:bg-transparent active:bg-transparent'
          asChild
        >
          <div className='flex items-center justify-between gap-2'>
            <span className='flex min-w-0 items-center gap-1.5 font-bold'>
              <Logo className='size-5 shrink-0 rounded-md' />
              <img
                alt='Pixoma'
                className={cn(
                  'h-[14px] w-auto shrink-0',
                  theme === 'dark' && 'invert'
                )}
                src='/images/Pixoma.svg'
              />
            </span>
            <ToggleSidebar />
          </div>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

function ToggleSidebar({
  className,
  onClick,
  ...props
}: React.ComponentProps<typeof Button>) {
  const { toggleSidebar } = useSidebar()

  return (
    <Button
      data-sidebar='trigger'
      data-slot='sidebar-trigger'
      variant='ghost'
      size='icon'
      className={cn('aspect-square size-8 max-md:scale-125', className)}
      onClick={(event) => {
        onClick?.(event)
        toggleSidebar()
      }}
      {...props}
    >
      <X className='md:hidden' />
      <Menu className='max-md:hidden' />
      <span className='sr-only'>Toggle Sidebar</span>
    </Button>
  )
}

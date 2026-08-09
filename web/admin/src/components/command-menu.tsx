import React from 'react'
import { useNavigate } from '@tanstack/react-router'
import { ArrowRight, Laptop, Moon, Sun } from 'lucide-react'
import { MENU_ITEMS } from '@/config/menu'
import { useSearch } from '@/context/search-provider'
import { useTheme } from '@/context/theme-provider'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { ScrollArea } from './ui/scroll-area'

/** Temporary visible labels until Task 5 wires i18n via titleKey. */
const TEMP_LABELS: Record<string, string> = {
  dashboard: 'Dashboard',
  instances: 'Instances',
  cases: 'Cases',
  tasks: 'Tasks',
  users: 'Users',
  sessions: 'Sessions',
}

export function CommandMenu() {
  const navigate = useNavigate()
  const { setTheme } = useTheme()
  const { open, setOpen } = useSearch()

  const runCommand = React.useCallback(
    (command: () => unknown) => {
      setOpen(false)
      command()
    },
    [setOpen]
  )

  return (
    <CommandDialog modal open={open} onOpenChange={setOpen}>
      <CommandInput placeholder='Type a command or search...' />
      <CommandList>
        <ScrollArea type='hover' className='h-72 pe-1'>
          <CommandEmpty>No results found.</CommandEmpty>
          <CommandGroup heading='Navigation'>
            {MENU_ITEMS.map((item) => {
              const label = TEMP_LABELS[item.id] ?? item.id
              return (
                <CommandItem
                  key={item.id}
                  value={label}
                  onSelect={() => {
                    runCommand(() => navigate({ to: item.path }))
                  }}
                >
                  <div className='flex size-4 items-center justify-center'>
                    <ArrowRight className='size-2 text-muted-foreground/80' />
                  </div>
                  {label}
                </CommandItem>
              )
            })}
          </CommandGroup>
          <CommandSeparator />
          <CommandGroup heading='Theme'>
            <CommandItem onSelect={() => runCommand(() => setTheme('light'))}>
              <Sun /> <span>Light</span>
            </CommandItem>
            <CommandItem onSelect={() => runCommand(() => setTheme('dark'))}>
              <Moon className='scale-90' />
              <span>Dark</span>
            </CommandItem>
            <CommandItem onSelect={() => runCommand(() => setTheme('system'))}>
              <Laptop />
              <span>System</span>
            </CommandItem>
          </CommandGroup>
        </ScrollArea>
      </CommandList>
    </CommandDialog>
  )
}

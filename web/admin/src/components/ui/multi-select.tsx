import * as React from 'react'
import { Check, Plus } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/lib/utils'

type MultiSelectProps = {
  value: string[]
  onValueChange: (next: string[]) => void
  placeholder?: string
  inputPlaceholder?: string
  createLabel?: string
  emptyText?: string
  disabled?: boolean
}

/**
 * shadcn Command + Popover 组合的多选（标签/分类）：
 * 支持搜索已有项、点选切换选中态、输入新值回车/点击新增。
 */
export function MultiSelect({
  value,
  onValueChange,
  placeholder,
  inputPlaceholder,
  createLabel,
  emptyText,
  disabled,
}: MultiSelectProps) {
  const [open, setOpen] = React.useState(false)
  const [query, setQuery] = React.useState('')

  const values = value.filter(Boolean)
  const selection = new Set(values)
  const q = query.trim().toLowerCase()

  const visible = values.filter((v) => !q || v.toLowerCase().includes(q))
  const canCreate = q.length > 0 && !values.some((v) => v.toLowerCase() === q)

  function toggle(next: string) {
    const trimmed = next.trim()
    if (!trimmed) return
    onValueChange(
      selection.has(trimmed)
        ? values.filter((v) => v !== trimmed)
        : [...values, trimmed]
    )
    setQuery('')
  }

  return (
    <Popover open={open} onOpenChange={(o) => {
      setOpen(o)
      if (!o) setQuery('')
    }}>
      <PopoverTrigger asChild>
        <Button
          type='button'
          variant='outline'
          disabled={disabled}
          className='h-auto min-h-9 w-full flex-wrap items-center justify-start gap-1 px-2 py-1.5 font-normal'
        >
          {values.length ? (
            values.map((v) => (
              <Badge
                key={v}
                variant='secondary'
                className='rounded-sm px-2 py-0 font-normal'
                data-testid='multi-select-badge'
              >
                {v}
              </Badge>
            ))
          ) : (
            <span className='text-muted-foreground'>{placeholder}</span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className='w-64 p-0' align='start'>
        <Command shouldFilter={false}>
          <CommandInput
            value={query}
            onValueChange={setQuery}
            placeholder={inputPlaceholder ?? placeholder}
          />
          <CommandList>
            {values.length === 0 && !canCreate ? (
              <CommandEmpty>{emptyText}</CommandEmpty>
            ) : null}
            <CommandGroup>
              {visible.map((v) => (
                <CommandItem key={v} value={v} onSelect={() => toggle(v)}>
                  <span
                    className={cn(
                      'flex size-4 items-center justify-center rounded-sm border border-primary',
                      selection.has(v)
                        ? 'bg-primary text-primary-foreground'
                        : 'opacity-40 [&_svg]:invisible'
                    )}
                  >
                    <Check className='size-3.5' />
                  </span>
                  <span className='min-w-0 flex-1 truncate'>{v}</span>
                </CommandItem>
              ))}
            </CommandGroup>
            {canCreate ? (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem value={query} onSelect={() => toggle(query)}>
                    <Plus className='size-4' />
                    {createLabel}
                    <span className='font-mono'>“{query.trim()}”</span>
                  </CommandItem>
                </CommandGroup>
              </>
            ) : null}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}

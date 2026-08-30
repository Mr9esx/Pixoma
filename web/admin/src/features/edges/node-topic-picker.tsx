import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { CheckIcon, ChevronsUpDown, Info } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { queryKeys } from '@/lib/api/query-keys'
import { listTopics, type Topic } from '@/lib/api/topics'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription } from '@/components/ui/alert'
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
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'

type Props = {
  /** 当前选中的 Topic key 集合（受控）。 */
  value: string[]
  /** 选中变化时回调。 */
  onChange: (next: string[]) => void
  /** 表单字段 id（与 Label htmlFor 配对）。 */
  id?: string
  /** 字段标签，默认 edges.subscribeTopics。 */
  label?: string
  /** 提示文案，默认 edges.subscribeTopicsHint。 */
  hint?: string
  /** 控件是否禁用（表单 pending 时）。 */
  disabled?: boolean
  /** 拉取的 Topic 列表（默认从 listTopics() 取）。 */
  topics?: Topic[]
  /** 是否包含未启用的 Topic（默认 false，仅展示启用的）。 */
  includeDisabled?: boolean
}

export function NodeTopicPicker({
  value,
  onChange,
  id = 'node-subscribe-topics',
  label,
  hint,
  disabled = false,
  topics: providedTopics,
  includeDisabled = false,
}: Props) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  const fallbackQuery = useQuery({
    queryKey: queryKeys.topics.all,
    queryFn: () => listTopics(),
    enabled: providedTopics === undefined,
  })
  const allTopics = providedTopics ?? fallbackQuery.data ?? []
  const topics = includeDisabled
    ? allTopics
    : allTopics.filter((tp) => tp.enabled)

  const resolvedLabel = label ?? t('edges.subscribeTopics')
  const resolvedHint = hint ?? t('edges.subscribeTopicsHint')
  const selectedLabels = value
    .map((key) => topics.find((tp) => tp.key === key)?.name || key)
    .join(', ')

  function toggle(key: string) {
    onChange(
      value.includes(key) ? value.filter((k) => k !== key) : [...value, key]
    )
  }
  function clear() {
    onChange([])
  }

  return (
    <div className='flex flex-col gap-2' data-testid='node-topic-picker'>
      <Label htmlFor={id}>{resolvedLabel}</Label>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            id={id}
            type='button'
            variant='outline'
            role='combobox'
            disabled={disabled}
            aria-expanded={open}
            className='w-full justify-between font-normal'
          >
            <span className='truncate'>
              {value.length > 0
                ? selectedLabels
                : t('edges.subscribeTopicsPlaceholder')}
            </span>
            <ChevronsUpDown className='ml-2 size-4 shrink-0 opacity-50' />
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className='w-(--radix-popover-trigger-width) p-0'
          align='start'
        >
          <Command>
            <CommandInput placeholder={t('edges.subscribeTopicsSearch')} />
            <CommandList>
              <CommandEmpty>{t('edges.subscribeTopicsEmpty')}</CommandEmpty>
              <CommandGroup>
                {topics.map((tp) => {
                  const selected = value.includes(tp.key)
                  return (
                    <CommandItem
                      key={tp.key}
                      value={tp.key}
                      onSelect={() => toggle(tp.key)}
                    >
                      <div
                        className={cn(
                          'flex size-4 items-center justify-center rounded-sm border border-primary',
                          selected
                            ? 'bg-primary text-primary-foreground'
                            : 'opacity-50 [&_svg]:invisible'
                        )}
                      >
                        <CheckIcon className='size-3 text-background' />
                      </div>
                      <span className='truncate font-medium'>
                        {tp.name || tp.key}
                      </span>
                      {tp.name ? (
                        <span className='truncate text-xs text-muted-foreground'>
                          {tp.key}
                        </span>
                      ) : null}
                    </CommandItem>
                  )
                })}
              </CommandGroup>
              {value.length > 0 ? (
                <>
                  <CommandSeparator />
                  <CommandGroup>
                    <CommandItem
                      onSelect={() => clear()}
                      className='justify-center text-center'
                    >
                      {t('edges.subscribeTopicsClear')}
                    </CommandItem>
                  </CommandGroup>
                </>
              ) : null}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
      <Alert variant='info'>
        <Info aria-hidden />
        <AlertDescription>{resolvedHint}</AlertDescription>
      </Alert>
    </div>
  )
}

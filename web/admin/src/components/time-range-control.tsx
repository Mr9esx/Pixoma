import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Calendar } from '@/components/ui/calendar'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatDate } from '@/features/dashboard/date-range'

export type RangePreset = {
  value: string
  label: string
  days: number
}

export function TimeRangeControl({
  presets,
  from,
  to,
  onChange,
}: {
  presets: RangePreset[]
  from: string
  to: string
  onChange: (range: { from: string; to: string }) => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [isCustom, setIsCustom] = useState(false)
  const span = betweenDays(from, to)
  const activePreset = presets.find((p) => p.days === span)

  return (
    <div className='flex flex-wrap items-center gap-2'>
      <Tabs
        value={activePreset?.value ?? 'custom'}
        onValueChange={(v) => {
          const preset = presets.find((p) => p.value === v)
          if (!preset) return
          const now = new Date()
          const end = now
          const start = new Date(now)
          start.setDate(start.getDate() - (preset.days - 1))
          onChange({ from: formatDate(start), to: formatDate(end) })
          setIsCustom(false)
        }}
      >
        <TabsList
          className='h-7 w-fit'
          aria-label={t('dashboard.workbench.range')}
        >
          {presets.map((p) => (
            <TabsTrigger
              key={p.value}
              value={p.value}
              className='h-full px-2.5 text-xs'
            >
              {p.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <Popover open={open} onOpenChange={(next) => setOpen(next)}>
        <div className='inline-flex h-7 items-center rounded-lg bg-muted p-0.75 text-muted-foreground'>
          <PopoverTrigger asChild>
            <button
              type='button'
              aria-label={t('dashboard.workbench.range')}
              className={cn(
                'inline-flex h-full items-center gap-1.5 rounded-md border border-transparent px-2.5 text-xs font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-1 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50',
                isCustom
                  ? 'border-input bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {t('dashboard.workbench.custom')}
              {isCustom ? (
                <span className='font-normal text-muted-foreground'>
                  <span className='tabular-nums'>
                    {from} ~ {to}
                  </span>
                </span>
              ) : null}
            </button>
          </PopoverTrigger>
        </div>
        <PopoverContent align='end' className='w-auto p-0'>
          <Calendar
            mode='range'
            defaultMonth={new Date(`${from}T00:00:00`)}
            selected={{
              from: new Date(`${from}T00:00:00`),
              to: new Date(`${to}T00:00:00`),
            }}
            onSelect={(sel) => {
              if (sel?.from && sel?.to) {
                onChange({
                  from: formatDate(sel.from),
                  to: formatDate(sel.to),
                })
                setIsCustom(true)
                setOpen(false)
              }
            }}
          />
        </PopoverContent>
      </Popover>
    </div>
  )
}

function betweenDays(from: string, to: string): number {
  const f = new Date(`${from}T00:00:00`).getTime()
  const t = new Date(`${to}T00:00:00`).getTime()
  return Math.round((t - f) / 86400000) + 1
}

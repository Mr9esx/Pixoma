import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { DAILY_PRESETS, daysAgo, formatDate } from './date-range'

export type StatsRange = { from: string; to: string }

export function TaskRangePicker({
  range,
  onChange,
}: {
  range: StatsRange
  onChange: (r: StatsRange) => void
}) {
  const { t } = useTranslation()
  return (
    <div className='flex flex-wrap items-center gap-2'>
      {DAILY_PRESETS.map((p) => (
        <Button
          key={p.labelKey}
          type='button'
          variant='outline'
          size='sm'
          onClick={() => onChange({ from: daysAgo(p.days - 1), to: daysAgo(0) })}
        >
          {t(p.labelKey)}
        </Button>
      ))}
      <Popover>
        <PopoverTrigger asChild>
          <Button type='button' variant='outline' size='sm'>
            {range.from} ~ {range.to}
          </Button>
        </PopoverTrigger>
        <PopoverContent align='end' className='w-auto p-0'>
          <Calendar
            mode='range'
            defaultMonth={new Date()}
            selected={{
              from: new Date(`${range.from}T00:00:00`),
              to: new Date(`${range.to}T00:00:00`),
            }}
            onSelect={(sel) => {
              if (sel?.from && sel?.to) {
                onChange({ from: formatDate(sel.from), to: formatDate(sel.to) })
              }
            }}
          />
        </PopoverContent>
      </Popover>
    </div>
  )
}

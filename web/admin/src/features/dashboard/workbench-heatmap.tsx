import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { eachDayOfInterval, formatISO, parseISO } from 'date-fns'
import { cn } from '@/lib/utils'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { Activity } from './daily-to-activity'

const LEVEL_CLASS = [
  'bg-muted',
  'bg-muted-foreground/20',
  'bg-muted-foreground/40',
  'bg-muted-foreground/60',
  'bg-muted-foreground/80',
]

export function WorkbenchHeatmap({ data }: { data: Activity[] }) {
  const { t } = useTranslation()
  const weeks = useMemo(() => groupByWeeks(data), [data])

  if (data.length === 0) return null

  return (
    <TooltipProvider delayDuration={0}>
      <div className='flex w-full flex-col gap-2'>
        <div className='flex w-full gap-2'>
          <div className='grid flex-1 grid-flow-col grid-rows-7 gap-1'>
            {weeks.map((week, weekIndex) =>
              week.map((activity, dayIndex) => {
                if (!activity) {
                  return <div key={`${weekIndex}-${dayIndex}`} />
                }
                return (
                  <Tooltip key={`${weekIndex}-${dayIndex}`}>
                    <TooltipTrigger asChild>
                      <div
                        className={cn(
                          'aspect-square w-full rounded-[2px]',
                          LEVEL_CLASS[activity.level] ?? LEVEL_CLASS[0]
                        )}
                        data-date={activity.date}
                        data-count={activity.count}
                      />
                    </TooltipTrigger>
                    <TooltipContent side='top' align='center'>
                      <p className='text-xs font-medium'>{activity.date}</p>
                      <p className='text-xs text-muted-foreground'>
                        {activity.count} {t('dashboard.workbench.taskCount')}
                      </p>
                    </TooltipContent>
                  </Tooltip>
                )
              })
            )}
          </div>
        </div>
      </div>
    </TooltipProvider>
  )
}

type Week = Array<Activity | undefined>

function fillHoles(activities: Activity[]): Activity[] {
  if (activities.length === 0) return []
  const sorted = [...activities].sort((a, b) => a.date.localeCompare(b.date))
  const byDate = new Map(sorted.map((a) => [a.date, a]))
  const first = parseISO(sorted[0].date)
  const last = parseISO(sorted[sorted.length - 1].date)
  return eachDayOfInterval({ start: first, end: last }).map((day) => {
    const date = formatISO(day, { representation: 'date' })
    return byDate.get(date) ?? { date, count: 0, level: 0 }
  })
}

function groupByWeeks(activities: Activity[]): Week[] {
  const sorted = [...activities].sort((a, b) => a.date.localeCompare(b.date))
  if (sorted.length === 0) return []
  const first = parseISO(sorted[0].date)
  const dow = first.getDay()
  const start = new Date(first)
  start.setDate(start.getDate() - dow)
  const days = fillHoles(sorted)
  const padded: Array<Activity | undefined> = []
  for (let i = 0; i < dow; i++) padded.push(undefined)
  padded.push(...days)
  const weeks: Week[] = []
  for (let i = 0; i < padded.length; i += 7) {
    weeks.push(padded.slice(i, i + 7))
  }
  return weeks
}

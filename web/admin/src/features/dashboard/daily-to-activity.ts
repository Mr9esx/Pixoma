import type { TaskDailyStat } from '@/lib/api/types'

export type Activity = {
  date: string
  count: number
  level: number
}

export function dailyToActivity(days: TaskDailyStat[]): Activity[] {
  if (days.length === 0) return []
  const max = Math.max(...days.map((d) => d.processed))
  return days.map((d) => {
    const count = d.processed
    const level = max <= 0 ? 0 : Math.min(4, Math.round((count / max) * 4))
    return { date: d.date, count, level }
  })
}

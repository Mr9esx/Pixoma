import { describe, expect, it } from 'vitest'
import { dailyToActivity } from './daily-to-activity'
import type { TaskDailyStat } from '@/lib/api/types'

function day(date: string, processed: number): TaskDailyStat {
  return {
    date,
    processed,
    succeeded: 0,
    failed: 0,
    cancelled: 0,
    avg_duration_ms: null,
    avg_queue_ms: null,
    avg_exec_ms: null,
  }
}

describe('dailyToActivity', () => {
  it('maps processed to count and level, with max day at level 4', () => {
    const days = [
      day('2026-08-01', 0),
      day('2026-08-02', 30),
      day('2026-08-03', 60),
      day('2026-08-04', 90),
      day('2026-08-05', 120),
    ]
    const acts = dailyToActivity(days)
    expect(acts).toHaveLength(5)
    expect(acts[0]).toMatchObject({ date: '2026-08-01', count: 0, level: 0 })
    expect(acts[4]).toMatchObject({ date: '2026-08-05', count: 120, level: 4 })
    expect(acts[1].level).toBeGreaterThan(0)
  })

  it('returns empty array for empty input', () => {
    expect(dailyToActivity([])).toEqual([])
  })

  it('keeps all levels at 0 when all processed counts are zero', () => {
    const acts = dailyToActivity([day('2026-08-01', 0), day('2026-08-02', 0)])
    expect(acts.every((a) => a.level === 0)).toBe(true)
  })
})

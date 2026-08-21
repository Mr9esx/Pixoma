import { describe, expect, it } from 'vitest'
import { DAILY_PRESETS, daysAgo, formatDate } from './date-range'

describe('date-range', () => {
  it('formats local date as YYYY-MM-DD', () => {
    expect(formatDate(new Date(2026, 7, 21))).toBe('2026-08-21')
    expect(formatDate(new Date(2026, 0, 5))).toBe('2026-01-05')
  })

  it('daysAgo returns today for 0 and a past date for n', () => {
    expect(daysAgo(0)).toBe(formatDate(new Date()))
    const past = daysAgo(30)
    expect(past).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    expect(past < daysAgo(0)).toBe(true)
  })

  it('exposes 7/30/90 day presets', () => {
    expect(DAILY_PRESETS.map((p) => p.days)).toEqual([7, 30, 90])
  })
})

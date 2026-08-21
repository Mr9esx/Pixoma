import { describe, expect, it } from 'vitest'
import {
  pickDays,
  pickEdgeItems,
  pickErrorItems,
  pickSuccessRate,
} from './task-stats-parse'

describe('task-stats-parse', () => {
  it('pickDays returns [] for undefined, non-object, or missing days', () => {
    expect(pickDays(undefined)).toEqual([])
    expect(pickDays('<!doctype html><html></html>')).toEqual([])
    expect(pickDays({ range: { from: 'a', to: 'b' } })).toEqual([])
  })

  it('pickDays returns days when present', () => {
    const days = [{ date: '2026-08-21', processed: 1, succeeded: 1, failed: 0, cancelled: 0, avg_duration_ms: null }]
    expect(pickDays({ days })).toEqual(days)
  })

  it('pickSuccessRate returns null for malformed or missing summary', () => {
    expect(pickSuccessRate(undefined)).toBeNull()
    expect(pickSuccessRate('html')).toBeNull()
    expect(pickSuccessRate({})).toBeNull()
    expect(pickSuccessRate({ summary: {} })).toBeNull()
    expect(pickSuccessRate({ summary: { success_rate: 'x' } })).toBeNull()
  })

  it('pickSuccessRate returns the number when present', () => {
    expect(pickSuccessRate({ summary: { success_rate: 0.5 } })).toBe(0.5)
  })

  it('pickErrorItems and pickEdgeItems tolerate malformed payloads', () => {
    expect(pickErrorItems(undefined)).toEqual([])
    expect(pickErrorItems({ items: 'nope' })).toEqual([])
    expect(pickErrorItems({ items: [{ error_code: 'timeout', count: 1 }] })).toEqual([
      { error_code: 'timeout', count: 1 },
    ])
    expect(pickEdgeItems({ items: [{ edge_id: 'gpu-1', count: 2 }] })).toEqual([
      { edge_id: 'gpu-1', count: 2 },
    ])
  })
})

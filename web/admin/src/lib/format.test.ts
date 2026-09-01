import { describe, expect, it } from 'vitest'
import { formatDateTime, formatUserLabel } from './format'

describe('formatDateTime', () => {
  it('converts UTC timestamps to the local timezone', () => {
    expect(formatDateTime('2026-08-31T12:00:00.000Z')).toMatch(
      /^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}$/
    )
  })

  it('hides zero-valued timestamps', () => {
    expect(formatDateTime('0001-01-01T00:00:00Z')).toBe('—')
    expect(formatDateTime(null)).toBe('—')
  })
})

describe('formatUserLabel', () => {
  it('prefers username, then composed name, then fallback', () => {
    expect(formatUserLabel({ username: 'alice' })).toBe('alice')
    expect(formatUserLabel({ first_name: 'Alice', last_name: 'A' })).toBe(
      'Alice A'
    )
    expect(formatUserLabel(null, 'user-1')).toBe('user-1')
  })
})

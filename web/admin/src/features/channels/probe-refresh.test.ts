import { describe, expect, it } from 'vitest'
import { captureProbeStamps, probeHasSettled } from './probe-refresh'

describe('probeHasSettled', () => {
  it('is not settled while last_check_at is unchanged after kick', () => {
    const before = captureProbeStamps([
      { id: 'tg', enabled: true, last_check_at: '2026-09-07T00:00:00Z' },
    ])
    expect(
      probeHasSettled(before, [
        { id: 'tg', enabled: true, last_check_at: '2026-09-07T00:00:00Z' },
      ])
    ).toBe(false)
  })

  it('settles when last_check_at changes', () => {
    const before = captureProbeStamps([
      { id: 'tg', enabled: true, last_check_at: '2026-09-07T00:00:00Z' },
    ])
    expect(
      probeHasSettled(before, [
        { id: 'tg', enabled: true, last_check_at: '2026-09-07T00:00:08Z' },
      ])
    ).toBe(true)
  })

  it('is not settled when last_check is still empty', () => {
    const before = captureProbeStamps([{ id: 'tg', enabled: true }])
    expect(probeHasSettled(before, [{ id: 'tg', enabled: true }])).toBe(false)
  })
})

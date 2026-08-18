import { describe, expect, it } from 'vitest'
import { formatBytes, parseMetrics } from './observation'

describe('parseMetrics', () => {
  it('maps optional gpu and io fields', () => {
    const out = parseMetrics({
      latest: {
        cpu_usage_percent: 30,
        mem_used_bytes: 8 * 1024 ** 3,
        mem_total_bytes: 16 * 1024 ** 3,
        mem_usage_percent: 50,
        gpus: [
          {
            name: 'RTX 4090',
            usage_percent: 60,
            vram_used_bytes: 12 * 1024 ** 3,
            vram_total_bytes: 24 * 1024 ** 3,
          },
        ],
        disk_read_bytes_per_sec: 1024,
        collected_at: '2026-08-18T12:00:00Z',
      },
      series: [],
    })
    expect(out.latest?.cpu).toBe(30)
    expect(out.latest?.gpus[0]?.name).toBe('RTX 4090')
    expect(out.latest?.ioRead).toBe(1024)
    expect(out.series).toEqual([])
  })

  it('defaults missing gpu and io to null', () => {
    const out = parseMetrics({
      latest: {
        cpu_usage_percent: 10,
        mem_used_bytes: 1,
        mem_total_bytes: 2,
        mem_usage_percent: 50,
        collected_at: '2026-08-18T12:00:00Z',
      },
      series: [],
    })
    expect(out.latest?.gpus).toEqual([])
    expect(out.latest?.ioRead).toBeNull()
    expect(out.latest?.ioWrite).toBeNull()
  })

  it('returns empty state for garbage', () => {
    const out = parseMetrics(null)
    expect(out.latest).toBeNull()
    expect(out.series).toEqual([])
  })

  it('formats bytes for charts', () => {
    expect(formatBytes(1024 ** 3)).toBe('1.0 GiB')
    expect(formatBytes(2 * 1024 ** 2)).toBe('2.0 MiB')
    expect(formatBytes(500)).toBe('500 B')
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(12 * 1024)).toBe('12.0 KiB')
  })
})

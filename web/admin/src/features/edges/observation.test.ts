import { describe, expect, it } from 'vitest'
import {
  formatBytes,
  formatMetricValue,
  ioAxisTicks,
  parseMetrics,
  seriesStats,
} from './observation'

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

describe('ioAxisTicks', () => {
  it('steps in KiB with clean whole-unit ticks', () => {
    const { ticks, format } = ioAxisTicks([524285, 102400])
    expect(ticks.length).toBeGreaterThanOrEqual(4)
    expect(ticks.every((t) => t >= 0)).toBe(true)
    expect(format(ticks[1])).toMatch(/^\d+ KiB$/)
    expect(format(0)).toBe('0 KiB')
  })

  it('steps in MiB for larger rates', () => {
    const { ticks, format } = ioAxisTicks([52_428_800])
    expect(format(ticks[1])).toMatch(/^\d+ MiB$/)
  })

  it('falls back to bytes for tiny rates', () => {
    const { ticks, format } = ioAxisTicks([300])
    expect(format(ticks[1])).toBe('100 B')
  })

  it('empty data yields zero tick', () => {
    const { ticks, format } = ioAxisTicks([])
    expect(ticks).toEqual([0])
    expect(format(0)).toBe('0 B')
  })
})

describe('formatMetricValue', () => {
  it('formats byte series with human units', () => {
    expect(formatMetricValue(17179869184, 'used')).toBe('16.0 GiB')
    expect(formatMetricValue(17179869184, 'gpu0Used')).toBe('16.0 GiB')
    expect(formatMetricValue(102400, 'read')).toBe('100 KiB')
  })

  it('formats percent series with %', () => {
    expect(formatMetricValue(42.5, 'cpu')).toBe('42.5%')
    expect(formatMetricValue(0, 'rate')).toBe('0.0%')
  })
})

describe('seriesStats', () => {
  it('returns current, max and average with the formatter', () => {
    const stats = seriesStats([10, 20, 15, 30], (v) => `${v.toFixed(1)}%`)
    expect(stats.current).toBe('30.0%')
    expect(stats.max).toBe('30.0%')
    expect(stats.avg).toBe('18.8%')
  })

  it('formats byte series with human units', () => {
    const stats = seriesStats([1024, 4 * 1024 ** 2], (v) => formatBytes(v))
    expect(stats.current).toBe('4.0 MiB')
    expect(stats.max).toBe('4.0 MiB')
    expect(stats.avg).toBe('2.0 MiB')
  })

  it('ignores null and non-finite values and falls back to dash when empty', () => {
    const stats = seriesStats([null, NaN, Infinity, 42, 48], (v) => `${v}%`)
    expect(stats.current).toBe('48%')
    expect(stats.max).toBe('48%')
    expect(stats.avg).toBe('45%')
    const empty = seriesStats([null, null], (v) => `${v}%`)
    expect(empty.current).toBe('—')
    expect(empty.max).toBe('—')
    expect(empty.avg).toBe('—')
  })
})

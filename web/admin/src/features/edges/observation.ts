import type { EdgeGPUMetric, EdgeMetrics } from '@/lib/api/types'

export type MetricsPoint = {
  time: number
  cpu: number | null
  memUsed: number | null
  memTotal: number | null
  memPct: number | null
  gpus: EdgeGPUMetric[]
  ioRead: number | null
  ioWrite: number | null
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function asMetrics(value: unknown): EdgeMetrics | null {
  return isRecord(value) ? (value as unknown as EdgeMetrics) : null
}

function toPoint(m: EdgeMetrics): MetricsPoint {
  return {
    time: Date.parse(m.collected_at),
    cpu: typeof m.cpu_usage_percent === 'number' ? m.cpu_usage_percent : null,
    memUsed: typeof m.mem_used_bytes === 'number' ? m.mem_used_bytes : null,
    memTotal: typeof m.mem_total_bytes === 'number' ? m.mem_total_bytes : null,
    memPct: typeof m.mem_usage_percent === 'number' ? m.mem_usage_percent : null,
    gpus: Array.isArray(m.gpus)
      ? (m.gpus.filter((gpu) => isRecord(gpu)) as EdgeGPUMetric[])
      : [],
    ioRead:
      typeof m.disk_read_bytes_per_sec === 'number'
        ? m.disk_read_bytes_per_sec
        : null,
    ioWrite:
      typeof m.disk_write_bytes_per_sec === 'number'
        ? m.disk_write_bytes_per_sec
        : null,
  }
}

export function parseMetrics(
  data: unknown
): { latest: MetricsPoint | null; series: MetricsPoint[] } {
  if (!isRecord(data)) return { latest: null, series: [] }
  const latest = asMetrics(data.latest)
  const series = Array.isArray(data.series)
    ? data.series.flatMap((item) => {
        const m = asMetrics(item)
        return m ? [toPoint(m)] : []
      })
    : []
  return { latest: latest ? toPoint(latest) : null, series }
}

export function formatBytes(n: number): string {
  const gib = 1024 ** 3
  const mib = 1024 ** 2
  if (Math.abs(n) >= gib) return `${(n / gib).toFixed(1)} GiB`
  return `${Math.round(n / mib)} MiB`
}

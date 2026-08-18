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
  if (!Number.isFinite(n) || n < 0) return '—'
  if (n === 0) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const idx = Math.min(
    Math.floor(Math.log(n) / Math.log(1024)),
    units.length - 1
  )
  const value = n / 1024 ** idx
  const text =
    idx === 0
      ? String(Math.round(value))
      : value >= 100
        ? value.toFixed(0)
        : value.toFixed(1)
  return `${text} ${units[idx]}`
}

// 图表 tooltip 数值格式化：字节序列走 formatBytes，其余按百分比展示。
export function formatMetricValue(value: unknown, key: string): string {
  const n = typeof value === 'number' ? value : Number(value)
  const isBytes =
    key === 'used' ||
    key === 'vramUsed' ||
    key === 'read' ||
    key === 'write'
  if (isBytes) {
    return Number.isFinite(n) ? formatBytes(n) : String(value ?? '')
  }
  return Number.isFinite(n) ? `${n.toFixed(1)}%` : String(value ?? '')
}

// I/O 轴阶梯化：按显示单位（B/KiB/MiB/GiB）用 1/2/5×10ⁿ 的整齐步进生成 tick，
// 避免小数值产生零碎小数（如 349.5 KiB、1.2 MiB）。
function niceTickStep(raw: number): number {
  if (raw <= 0) return 1
  const mag = 10 ** Math.floor(Math.log10(raw))
  const n = raw / mag
  const nice = n <= 1 ? 1 : n <= 2 ? 2 : n <= 5 ? 5 : 10
  return nice * mag
}

export function ioAxisTicks(values: number[]): {
  ticks: number[]
  format: (v: number) => string
} {
  const max = values.length ? Math.max(...values) : 0
  if (max <= 0) {
    return { ticks: [0], format: () => '0 B' }
  }
  let unit = 1
  let label = 'B'
  if (max >= 4 * 1024 ** 3) {
    unit = 1024 ** 3
    label = 'GiB'
  } else if (max >= 4 * 1024 ** 2) {
    unit = 1024 ** 2
    label = 'MiB'
  } else if (max >= 4 * 1024) {
    unit = 1024
    label = 'KiB'
  }
  const maxUnits = max / unit
  const step = niceTickStep(maxUnits / 4)
  const ticks: number[] = []
  for (let u = 0; u <= maxUnits + step && ticks.length < 7; u += step) {
    ticks.push(Math.round(u * unit))
  }
  const format =
    unit === 1
      ? (v: number) => `${Math.round(v)} B`
      : (v: number) => `${Math.round(v / unit)} ${label}`
  return { ticks, format }
}

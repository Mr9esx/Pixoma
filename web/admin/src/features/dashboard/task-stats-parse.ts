import type {
  TaskDailyStat,
  TaskEdgeStat,
  TaskErrorStat,
} from '@/lib/api/types'

function asObject(data: unknown): Record<string, unknown> | null {
  if (typeof data !== 'object' || data === null) return null
  return data as Record<string, unknown>
}

export function pickDays(data: unknown): TaskDailyStat[] {
  const obj = asObject(data)
  const days = obj?.days
  return Array.isArray(days) ? (days as TaskDailyStat[]) : []
}

export function pickSuccessRate(data: unknown): number | null {
  const obj = asObject(data)
  if (!obj) return null
  const summary = asObject(obj.summary)
  const rate = summary?.success_rate
  return typeof rate === 'number' && Number.isFinite(rate) ? rate : null
}

export function pickErrorItems(data: unknown): TaskErrorStat[] {
  const obj = asObject(data)
  const items = obj?.items
  return Array.isArray(items) ? (items as TaskErrorStat[]) : []
}

export function pickEdgeItems(data: unknown): TaskEdgeStat[] {
  const obj = asObject(data)
  const items = obj?.items
  return Array.isArray(items) ? (items as TaskEdgeStat[]) : []
}

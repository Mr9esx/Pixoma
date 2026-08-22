import { apiFetch, toQuery } from './client'
import type {
  FleetStats,
  TaskCaseTopResponse,
  TaskDailyStatsResponse,
  TaskEdgeStat,
  TaskErrorStat,
} from './types'

export function listTaskDailyStats(params: { from: string; to: string }) {
  return apiFetch<TaskDailyStatsResponse>(
    `/api/v1/stats/tasks/daily${toQuery(params)}`
  )
}

export function listTaskErrorStats(params: {
  from: string
  to: string
  limit?: number
}) {
  return apiFetch<{ items: TaskErrorStat[] }>(
    `/api/v1/stats/tasks/errors${toQuery(params)}`
  )
}

export function listTaskEdgeStats(params: { from: string; to: string }) {
  return apiFetch<{ items: TaskEdgeStat[]; total: number }>(
    `/api/v1/stats/tasks/edges${toQuery(params)}`
  )
}

export function listTaskCaseTopStats(params: {
  from: string
  to: string
  limit?: number
}) {
  return apiFetch<TaskCaseTopResponse>(
    `/api/v1/stats/cases/top${toQuery(params)}`
  )
}

export function listFleetStats() {
  return apiFetch<FleetStats>('/api/v1/stats/fleet')
}

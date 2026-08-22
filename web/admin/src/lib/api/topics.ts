import { apiFetch, toQuery } from './client'

export type Topic = {
  key: string
  name: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export type TopicErrorCodeCount = { code: string; count: number }
export type TopicRuntimeStats = { sum_ms: number; avg_ms: number | null; count: number }
export type TopicThroughputPoint = { ts: string; count: number }

export type TopicStats = {
  task_count: number
  status: Record<string, number>
  success_rate: number | null
  error_codes: TopicErrorCodeCount[]
  runtime_ms: TopicRuntimeStats
  throughput: TopicThroughputPoint[]
  from: string
  to: string
}

export function listTopics(enabled?: boolean) {
  return apiFetch<Topic[]>(`/api/v1/topics${toQuery({ enabled })}`)
}

export function createTopic(body: { key: string; name: string }) {
  return apiFetch<Topic>('/api/v1/topics', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getTopic(key: string) {
  return apiFetch<Topic>(`/api/v1/topics/${encodeURIComponent(key)}`)
}

export function updateTopic(
  key: string,
  body: { name?: string; enabled?: boolean },
) {
  return apiFetch<Topic>(`/api/v1/topics/${encodeURIComponent(key)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export function deleteTopic(key: string) {
  return apiFetch<void>(`/api/v1/topics/${encodeURIComponent(key)}`, {
    method: 'DELETE',
  })
}

export function getTopicStats(key: string) {
  return apiFetch<TopicStats>(`/api/v1/topics/${encodeURIComponent(key)}/stats`)
}

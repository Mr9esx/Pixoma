import { apiFetch, toQuery } from './client'

export type Topic = {
  key: string
  name: string
  enabled: boolean
  created_at: string
  updated_at: string
}

type TopicErrorCodeCount = { code: string; count: number }
type TopicRuntimeStats = {
  sum_ms: number
  avg_ms: number | null
  count: number
}
type TopicThroughputPoint = { ts: string; count: number }

type TopicStats = {
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
  body: { name?: string; enabled?: boolean }
) {
  return apiFetch<Topic>(`/api/v1/topics/${encodeURIComponent(key)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

type DeleteTopicResult = {
  deleted: boolean
  removed_case_rules?: number
  removed_edge_subscriptions?: number
  failed_tasks?: number
}

export function deleteTopic(key: string, ack?: boolean) {
  return apiFetch<DeleteTopicResult>(
    `/api/v1/topics/${encodeURIComponent(key)}`,
    {
      method: 'DELETE',
      ...(ack ? { body: JSON.stringify({ ack_references: true }) } : {}),
    }
  )
}

export function getTopicStats(key: string) {
  return apiFetch<TopicStats>(`/api/v1/topics/${encodeURIComponent(key)}/stats`)
}

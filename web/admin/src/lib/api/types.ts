export type EdgeHardwareGPU = {
  name: string
  vram_bytes?: number
}

export type EdgeHardware = {
  cpu_model?: string
  cpu_cores?: number
  ram_bytes?: number
  gpus?: EdgeHardwareGPU[]
  collected_at?: string
}

export type EdgeGPUMetric = {
  name: string
  usage_percent?: number | null
  vram_used_bytes?: number
  vram_total_bytes?: number
  vram_usage_percent?: number | null
}

export type EdgeMetrics = {
  cpu_usage_percent: number
  mem_used_bytes: number
  mem_total_bytes: number
  mem_usage_percent: number
  gpus?: EdgeGPUMetric[]
  disk_read_bytes_per_sec?: number | null
  disk_write_bytes_per_sec?: number | null
  collected_at: string
}

export type EdgeMetricsResponse = {
  latest: EdgeMetrics | null
  series: EdgeMetrics[]
}

export type ComfyEdge = {
  id: string
  name: string
  description?: string
  enabled: boolean
  capabilities: string[]
  agent_token?: string
  subscribe_topics?: string[]
  effective_topics?: string[]
  hardware?: EdgeHardware
  started_at: string | null
  comfy_version: string
  created_at: string
  updated_at: string
}

export type EdgeStats = {
  task_count: number
  runtime_ms: number
  success_rate: number | null
}

export type EdgePresence = {
  id: string
  edge_online: boolean
  comfy_running: boolean
}

export type CaseInputField = {
  key: string
  type: string
  required: boolean
  skip_allowed?: boolean
  description?: string
  preview?: string
}

export type CaseOutputField = {
  key: string
  type: string
  description?: string
  media_type?: string
}

export type InputBinding = { key: string; node_id: string; field_path: string }
export type OutputBinding = { key: string; node_id: string; index?: number }

export type CaseRecord = {
  id: number
  name: string
  description?: string
  preview?: string
  tags?: string[]
  categories?: string[]
  inputs: CaseInputField[]
  outputs: CaseOutputField[]
  bindings: {
    workflow: Record<string, unknown>
    inputs: InputBinding[]
    outputs: OutputBinding[]
  }
  input_schema: Record<string, unknown>
  workflow_filename?: string
  enabled: boolean
  routing?: RoutingConfig
}

/** Case 路由配置（与后端 CaseDocument.Routing 对齐）。 */
export type ConditionOp =
  | 'eq'
  | 'ne'
  | 'in'
  | 'gt'
  | 'gte'
  | 'lt'
  | 'lte'
  | 'exists'

export type RoutingCondition =
  | { field: string; op: ConditionOp; value?: unknown }
  | { and: RoutingCondition[] }
  | { or: RoutingCondition[] }

export type RoutingRule = {
  when: RoutingCondition
  topic?: string
}

export type RoutingConfig = {
  rules: RoutingRule[]
}

export type CaseWithRouting = CaseRecord & {
  routing?: RoutingConfig
}

export type TaskRecord = {
  id: string
  session_id: string
  chat_id?: number
  case_id: number
  status: string
  edge_id?: string
  dispatch_topic?: string
  prompt_id?: string
  error_code?: string
  error_message?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

export type TaskDailyStat = {
  date: string
  processed: number
  succeeded: number
  failed: number
  cancelled: number
  avg_duration_ms: number | null
  avg_queue_ms: number | null
  avg_exec_ms: number | null
}

export type TaskDailyStatsResponse = {
  range: { from: string; to: string }
  days: TaskDailyStat[]
  summary: {
    processed: number
    succeeded: number
    failed: number
    cancelled: number
    success_rate: number | null
  }
}

export type TaskErrorStat = { error_code: string; count: number }
export type TaskEdgeStat = {
  edge_id: string
  count: number
  succeeded: number
  failed: number
  success_rate: number | null
}

export type TaskCaseStat = {
  case_id: number
  count: number
  avg_duration_ms: number | null
}

export type TaskCaseTopResponse = { items: TaskCaseStat[] }

export type FleetNode = {
  edge_id: string
  cpu_usage_percent: number
  mem_usage_percent: number
  gpu_usage_percent?: number | null
  vram_used_bytes?: number
  vram_total_bytes?: number
}

export type FleetStats = {
  online: number
  avg_cpu_usage_percent: number
  avg_mem_usage_percent: number
  avg_gpu_usage_percent?: number | null
  vram_used_bytes: number
  vram_total_bytes: number
  hottest?: { edge_id: string; cpu_usage_percent: number } | null
  nodes: FleetNode[]
}

export type UserRecord = {
  id: string
  tg_user_id: number
  username: string
  first_name: string
  last_name: string
  language_code: string
  last_seen_at: string
  created_at: string
  updated_at: string
}

export type SessionDraft = {
  key: string
  text?: string
  number?: number
  bool?: boolean
  blob?: unknown
  skipped?: boolean
}

export type SessionRecord = {
  id: string
  user_id: string
  chat_id: number
  case_id: number
  status: string
  current_input_index: number
  input_keys: string[]
  draft: Record<string, SessionDraft>
  created_at: string
  updated_at: string
}

export type ListParams = {
  q?: string
  limit?: number
  offset?: number
  created_from?: string
  created_to?: string
  [key: string]: string | number | boolean | undefined
}

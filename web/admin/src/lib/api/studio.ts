import { apiFetch, baseURL, sessionToken, toQuery } from './client'

export type StudioPermissionMode =
  | 'request_approval'
  | 'auto_approve'
  | 'full_access'

export type StudioSession = {
  id: string
  project_id?: string
  title: string
  permission_mode: StudioPermissionMode
  model_config_id?: string
  status: 'active'
  latest_run?: StudioRun | null
  active_workflow_count?: number
  created_at: string
  updated_at: string
}

export type StudioProject = {
  id: string
  name: string
  created_at: string
  updated_at: string
}

export type StudioMessagePart = {
  type:
    | 'text'
    | 'image'
    | 'file'
    | 'reasoning'
    | 'skill_ref'
    | 'asset_ref'
    | 'workflow_ref'
  text?: string
  url?: string
  name?: string
  skill_id?: string
  asset_id?: string
  asset_version_id?: string
  workflow_id?: string
}

export type StudioComposerPart =
  | { type: 'text'; text: string }
  | { type: 'skill_ref'; skill_id: string; name: string }
  | {
      type: 'asset_ref'
      asset_id: string
      asset_version_id: string
      name: string
    }
  | { type: 'workflow_ref'; workflow_id: string; name: string }

export type StudioTranscriptToolCall = {
  id: string
  type: 'function'
  function: { name: string; arguments: string }
}

export type StudioTranscriptMessage = {
  id: string
  runId?: string
  role: 'user' | 'assistant' | 'reasoning' | 'tool'
  content: string
  parts?: StudioComposerPart[]
  toolCalls?: StudioTranscriptToolCall[]
  toolCallId?: string
  isError?: boolean
  clarification?: StudioTranscriptClarification
}

export type StudioTranscriptClarification = {
  id: string
  question: string
  options: string[]
  selected?: string
  answer?: string
  status: 'pending' | 'answered' | 'skipped'
  workflow?: StudioWorkflowRequest
}

export type StudioWorkflowInputField = {
  key: string
  type: string
  required: boolean
  description?: string
}

export type StudioWorkflowRequest = {
  id: string
  name: string
  description?: string
  preview?: string
  input_schema: Record<string, unknown>
  input_fields: StudioWorkflowInputField[]
  suggested_inputs?: Record<string, unknown>
  submitted_inputs?: Record<string, unknown>
}

export type StudioTranscriptEvent = {
  id: string
  runId: string
  sequence: number
  type: string
  payload: Record<string, unknown>
  createdAt: string
}

export type StudioTranscript = {
  messages: StudioTranscriptMessage[]
  events: StudioTranscriptEvent[]
}

export type StudioMessage = {
  id: string
  session_id: string
  run_id?: string
  role: 'user' | 'assistant' | 'system' | 'tool'
  content: StudioMessagePart[]
  created_at: string
}

export type StudioRun = {
  id: string
  session_id: string
  trigger_message_id: string
  status:
    | 'queued'
    | 'running'
    | 'waiting_approval'
    | 'waiting_clarification'
    | 'succeeded'
    | 'failed'
    | 'cancelled'
  model_config_id?: string
  error_code?: string
  error_message?: string
  created_at: string
  started_at?: string
  completed_at?: string
  updated_at: string
}

export type StudioRunProgress = {
  run_id: string
  session_id: string
  assistant_message_id?: string
  assistant_text?: string
  reasoning_text?: string
  tool_calls?: Record<
    string,
    {
      id: string
      name: string
      args?: string
      result?: string
      is_error?: boolean
    }
  >
  last_sequence: number
  updated_at: string
}

export type StudioRunEvent = {
  id: string
  run_id: string
  sequence: number
  type: string
  payload: Record<string, unknown>
  created_at: string
}

export type StudioTrajectoryRecord = {
  id: string
  run_id: string
  kind: string
  step?: number
  attempt?: number
  parent_id?: string
  title: string
  preview?: string
  status: string
  started_at: string
  ended_at?: string
}

export type StudioTrajectoryDetail = {
  record: StudioTrajectoryRecord
  overview: Record<string, unknown>
  input?: unknown
  output?: unknown
  raw?: unknown
  schema?: unknown
  usage?: Record<string, unknown>
  timing?: Record<string, unknown>
}

export type StudioTrajectoryPage = {
  runs: { run: StudioRun; records: StudioTrajectoryRecord[] }[]
  next_cursor: string
  has_more: boolean
  total_runs: number
}

export type StudioContextPart = {
  id: string
  category: string
  source: string
  label: string
  content?: string
  estimated_tokens: number
}

export type StudioContextRequest = {
  attempt_id: string
  session_id: string
  run_id: string
  turn_id: string
  turn_number: number
  step_number: number
  purpose: string
  protocol: string
  model: string
  preview: string
  status: string
  started_at: string
  first_token_at?: string
  ended_at?: string
  context_window_tokens: number
  max_input_tokens: number
  max_output_tokens: number
  input_tokens: number | null
  output_tokens: number | null
  cache_read_tokens: number | null
  cache_write_tokens: number | null
  reasoning_tokens: number | null
  projected_tokens?: number
  parts: StudioContextPart[]
  content_available?: boolean
}

export type StudioContextOverview = {
  turns: number
  steps: number
  tool_calls: number
  injections: number
  compactions: number
  prunes: number
  tokens: {
    input: number
    output: number
    cache_read: number
    cache_write: number
    uncached: number
    unclassified_input: number
    reasoning: number
    missing_requests: number
    cache_known_requests: number
    cache_known_input: number
  }
  timing: {
    active_ms: number
    model_wait_ms: number
    generation_ms: number
    model_other_ms: number
    tools_ms: number
    overlap_ms: number
    other_ms: number
  }
  current: StudioContextRequest | null
}

export type StudioContextEvent = {
  id: string
  run_id: string
  turn_number: number
  step_number: number
  kind: string
  source: string
  detail: string
  delta_tokens_estimated: number | null
  before_tokens_estimated: number | null
  after_tokens_estimated: number | null
  created_at: string
}

export type StudioAssetVersion = {
  id: string
  version: number
  mime_type: string
  size_bytes: number
  metadata?: Record<string, unknown>
  content_url: string
  created_at: string
  format?: string
  width_px?: number | null
  height_px?: number | null
  source_created_at?: string | null
  source_modified_at?: string | null
  content_origin?: string
  palette?: StudioAssetPalette | null
}

export type StudioAssetPalette = {
  status: 'pending' | 'running' | 'ready' | 'failed'
  colors: Array<{ hex: string; ratio: number }>
  analyzed_at?: string | null
  error_code?: string
}

export type StudioAsset = {
  id: string
  name: string
  kind: 'document' | 'image' | 'video' | 'audio' | 'data' | 'file'
  origin: 'user' | 'agent' | 'model' | 'workflow'
  current_version: number
  versions: StudioAssetVersion[]
  usages?: StudioSessionAssetUsage[]
  created_at: string
  updated_at: string
}

export type StudioLibraryCategory = {
  id: string
  project_id?: string
  parent_id?: string
  name: string
  created_at: string
  updated_at: string
}

export type StudioLibraryAssetsPage = {
  items: StudioProjectAsset[]
  total: number
  next_cursor?: string
}

export type StudioLibraryTreeMode =
  | 'asset'
  | 'session'
  | 'category'
  | 'format'
  | 'rating'
  | 'tag'

export type StudioLibraryProject = {
  id: string
  name: string
  asset_count: number
}

export type StudioLibraryTreeNode = {
  id: string
  label: string
  count: number
  group_value?: string
} & (
  | { kind: 'asset'; asset_id: string; asset_kind: StudioAsset['kind'] }
  | { kind: 'group'; asset_id?: never }
)

export type StudioLibraryTreePage = {
  nodes: StudioLibraryTreeNode[]
  next_cursor?: string
}

export type StudioAssetTag = {
  id: string
  name: string
}

export type StudioProjectAsset = {
  id: string
  project_id: string
  asset_id: string
  asset_version_id: string
  display_name: string
  category_id: string
  rating: number
  tags: StudioAssetTag[]
  added_at: string
  updated_at: string
  archived_at?: string | null
  asset: StudioAsset
  version: StudioAssetVersion
  copy_source?: {
    project_id: string
    project_asset_id: string
    project_name_snapshot: string
    display_name: string
    deleted: boolean
  } | null
}

export type StudioSessionAssetUsage = {
  id: string
  session_id: string
  session_title_snapshot: string
  usage_kind: 'created' | 'uploaded' | 'referenced'
  run_id?: string
  created_at: string
  session_available?: boolean
}

export type StudioProjectAssetDetail = StudioProjectAsset & {
  usages: StudioSessionAssetUsage[]
  versions: StudioAssetVersion[]
}

export type StudioLibraryPreferences = {
  tree_mode: StudioLibraryTreeMode
  last_project_id: string
}

export type StudioFlowNode = {
  id: string
  type: 'stage' | 'plan' | 'operation' | 'asset'
  title: string
  body?: string
  asset_id?: string
  asset_version_id?: string
  asset_version?: number
  run_id?: string
  outputs?: Array<{ key: string; type: string; name: string; asset_id: string; asset_version_id: string }>
  position: { x: number; y: number }
  sort_order: number
  updated_at: string
}

export type StudioFlowEdge = {
  id: string
  source: string
  target: string
  source_output_key?: string
  target_input_key?: string
  label?: string
}

export type StudioWorkflowExecution = {
  id: string
  run_id: string
  task_id: string
  workflow_id: string
  operation_node_id: string
  input_fields?: Array<{ key: string; type: string; description?: string; required?: boolean }>
  output_fields?: Array<{ key: string; type: string; description?: string; required?: boolean }>
  inputs?: Array<{ key: string; value?: string; asset_id?: string; asset_version_id?: string; asset_name?: string }>
  status: 'submitted' | 'succeeded' | 'failed' | 'cancelled'
  task_status?:
    | 'pending'
    | 'queued'
    | 'running'
    | 'succeeded'
    | 'failed'
    | 'cancelled'
  error_message?: string
  created_at: string
  completed_at?: string
}

// 运行处于等待批准时随会话详情返回的待处理事项，形状与 AG-UI interrupt 一致
export type StudioPendingApproval = {
  id: string
  reason?: string
  message?: string
}

export type StudioPendingClarification = {
  id: string
  reason: 'input_required' | 'workflow_input'
  message: string
  metadata: {
    options?: { id: string; label: string }[]
    messageId: string
    workflow?: StudioWorkflowRequest
  }
}

export type StudioSessionDetail = {
  session: StudioSession
  run_progress?: StudioRunProgress | null
  pending_approvals?: StudioPendingApproval[]
  pending_clarifications?: StudioPendingClarification[]
  messages: StudioMessage[]
  transcript: StudioTranscript
  workflow_executions?: StudioWorkflowExecution[]
  assets: StudioAsset[]
  flow: {
    nodes: StudioFlowNode[]
    edges: StudioFlowEdge[]
  }
}

export type StudioTurn = {
  session: StudioSession
  message: StudioMessage
  run: StudioRun
}

export type StudioModel = {
  id: string
  name: string
  protocol:
    | 'openai_responses'
    | 'openai_chat_compatible'
    | 'anthropic_messages_compatible'
  base_url: string
  model: string
  has_api_key: boolean
  api_key_masked?: string
  enabled: boolean
  agent_enabled: boolean
  default: boolean
  limits: {
    context_window_tokens: number
    max_input_tokens: number
    max_output_tokens: number
  }
  thinking: {
    enabled: boolean
    effort?: string
    budget_tokens?: number
  }
  capabilities: {
    tools: boolean
    vision: boolean
    image_output: boolean
    streaming: boolean
  }
}

export type StudioModelConnectionTest = {
  success: boolean
  latency_ms: number
}

export type StudioSkillSummary = {
  id: string
  name: string
  description: string
  version: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export type StudioSkill = StudioSkillSummary & {
  prompt: string
  files?: StudioSkillFile[]
}

export type StudioSkillFile = {
  path: string
  content: string
  binary?: boolean
  directory?: boolean
}

export type StudioSkillVersionSummary = {
  version: string
  created_at: string
}

export type StudioSkillVersion = StudioSkillVersionSummary & {
  name: string
  description: string
  prompt: string
  files: StudioSkillFile[]
}

export type StudioConnectorPolicy = 'auto' | 'approval' | 'forbidden'

export type StudioMCPTool = { name: string; description: string }

export type StudioMCPConnector = {
  id: string
  name: string
  url: string
  enabled: boolean
  policy: StudioConnectorPolicy
  credential_masked: string
  tools: StudioMCPTool[] | null
  created_at: string
  updated_at: string
}

export type StudioAgentWorkflow = {
  id: string
  name: string
  description: string
  workflow_enabled: boolean
  agent_enabled: boolean
  inputs: number
  outputs: number
  preview?: string
  input_schema: Record<string, unknown>
  input_fields: StudioWorkflowInputField[]
}

export function listStudioSessions(params?: {
  limit?: number
  offset?: number
  project_id?: string
}) {
  const query = toQuery(params)
  const unassigned =
    params?.project_id === '' ? `${query ? '&' : '?'}project_id=` : ''
  return apiFetch<StudioSession[]>(
    `/api/v1/studio/sessions${query}${unassigned}`
  )
}

function studioProject(value: unknown): StudioProject {
  if (
    typeof value !== 'object' ||
    value === null ||
    !('id' in value) ||
    typeof value.id !== 'string' ||
    !('name' in value) ||
    typeof value.name !== 'string'
  ) {
    throw new Error('项目响应格式错误')
  }
  return value as StudioProject
}

export async function listStudioProjects() {
  const projects = await apiFetch<unknown>('/api/v1/studio/projects')
  if (!Array.isArray(projects)) throw new Error('项目列表响应格式错误')
  return projects.map(studioProject)
}

export async function createStudioProject(name: string) {
  const project = await apiFetch<unknown>('/api/v1/studio/projects', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
  return studioProject(project)
}

export async function renameStudioProject(id: string, name: string) {
  const project = await apiFetch<unknown>(
    `/api/v1/studio/projects/${encodeURIComponent(id)}`,
    {
      method: 'PATCH',
      body: JSON.stringify({ name }),
    }
  )
  return studioProject(project)
}

export function deleteStudioProject(id: string) {
  return apiFetch<void>(`/api/v1/studio/projects/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}

export function moveStudioSessionToProject(
  sessionId: string,
  projectId: string
) {
  return apiFetch<StudioSession>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/project`,
    {
      method: 'PATCH',
      body: JSON.stringify({ project_id: projectId }),
    }
  )
}

export function createStudioSession(requestId: string) {
  return apiFetch<StudioSession>('/api/v1/studio/sessions', {
    method: 'POST',
    body: JSON.stringify({ request_id: requestId }),
  })
}

export function clearStudioSessions() {
  return apiFetch<void>('/api/v1/studio/sessions', {
    method: 'DELETE',
    body: JSON.stringify({ confirmation: '确认清空' }),
  })
}

export function getStudioSession(sessionId: string) {
  return apiFetch<StudioSessionDetail>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}`
  )
}

export function sendStudioMessage(input: {
  sessionId?: string
  projectId?: string
  requestId?: string
  text: string
  locale: 'zh' | 'en'
  parts?: StudioComposerPart[]
  modelConfigId?: string
  permissionMode: StudioPermissionMode
}) {
  return apiFetch<StudioTurn>('/api/v1/studio/messages', {
    method: 'POST',
    body: JSON.stringify({
      session_id: input.sessionId,
      project_id: input.projectId,
      request_id: input.requestId,
      text: input.text,
      locale: input.locale,
      parts: input.parts,
      model_config_id: input.modelConfigId,
      permission_mode: input.permissionMode,
    }),
  })
}

export function getStudioRun(runId: string) {
  return apiFetch<StudioRun>(`/api/v1/studio/runs/${encodeURIComponent(runId)}`)
}

export function cancelStudioRun(runId: string) {
  return apiFetch<void>(
    `/api/v1/studio/runs/${encodeURIComponent(runId)}/cancel`,
    {
      method: 'POST',
    }
  )
}

export function listStudioSessionRuns(sessionId: string) {
  return apiFetch<StudioRun[]>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/runs`
  )
}

export function listStudioRunEvents(runId: string) {
  return apiFetch<StudioRunEvent[]>(
    `/api/v1/studio/runs/${encodeURIComponent(runId)}/events`
  )
}

export function getStudioSessionTrajectory(sessionId: string, before?: string) {
  return apiFetch<StudioTrajectoryPage>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/trajectory${toQuery({ before })}`
  )
}

export function getStudioTrajectoryRecord(
  sessionId: string,
  runId: string,
  recordId: string
) {
  return apiFetch<StudioTrajectoryDetail>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/trajectory/runs/${encodeURIComponent(runId)}/records/${encodeURIComponent(recordId)}`
  )
}

export function getStudioSessionContext(sessionId: string) {
  return apiFetch<StudioContextOverview>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/context`
  )
}

export function getStudioCurrentContext(sessionId: string) {
  return apiFetch<StudioContextRequest>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/context/current`
  )
}

export function listStudioContextRequests(sessionId: string, offset = 0) {
  return apiFetch<{
    requests: StudioContextRequest[]
    has_more: boolean
    next_offset: number
  }>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/context/requests${toQuery({ offset })}`
  )
}

export function getStudioContextRequest(sessionId: string, attemptId: string) {
  return apiFetch<StudioContextRequest>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/context/requests/${encodeURIComponent(attemptId)}`
  )
}

export function listStudioContextEvents(sessionId: string, offset = 0, kind = '') {
  return apiFetch<{
    events: StudioContextEvent[]
    has_more: boolean
    next_offset: number
  }>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/context/events${toQuery({ offset, kind })}`
  )
}

export function resolveStudioApproval(approvalId: string, approved: boolean) {
  return apiFetch<void>(
    `/api/v1/studio/approvals/${encodeURIComponent(approvalId)}`,
    { method: 'POST', body: JSON.stringify({ approved }) }
  )
}

export function referenceStudioAsset(
  sessionId: string,
  assetId: string,
  assetVersionId: string,
  requestId: string,
  sourceProjectAssetId?: string
) {
  return apiFetch<StudioAsset>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/assets/references`,
    {
      method: 'POST',
      body: JSON.stringify({
        asset_id: assetId,
        asset_version_id: assetVersionId,
        request_id: requestId,
        ...(sourceProjectAssetId ? { source_project_asset_id: sourceProjectAssetId } : {}),
      }),
    }
  )
}

export function createStudioTextAsset(input: {
  sessionId: string
  name: string
  content: string
  requestId?: string
}) {
  return apiFetch<StudioAsset>('/api/v1/studio/assets/text', {
    method: 'POST',
    body: JSON.stringify({
      session_id: input.sessionId,
      name: input.name,
      content: input.content,
      request_id: input.requestId ?? crypto.randomUUID(),
    }),
  })
}

export function updateStudioTextAsset(
  assetId: string,
  content: string,
  requestId: string = crypto.randomUUID()
) {
  return apiFetch<StudioAsset>(
    `/api/v1/studio/assets/${encodeURIComponent(assetId)}/text`,
    { method: 'PATCH', body: JSON.stringify({ content, request_id: requestId }) }
  )
}

export function getStudioTextAssetContent(contentURL: string) {
  return apiFetch<string>(contentURL)
}

export function uploadStudioAsset(
  file: File,
  sessionId?: string,
  projectId = '',
  requestId: string = crypto.randomUUID()
) {
  const body = new FormData()
  body.append('file', file)
  if (sessionId) body.append('session_id', sessionId)
  else body.append('project_id', projectId)
  body.append('request_id', requestId)
  if (Number.isFinite(file.lastModified) && file.lastModified > 0)
    body.append('source_modified_at', new Date(file.lastModified).toISOString())
  return apiFetch<StudioAsset & { project_asset_id: string }>('/api/v1/studio/assets/upload', {
    method: 'POST',
    body,
  })
}

export function getStudioAsset(assetId: string) {
  return apiFetch<StudioAsset>(
    `/api/v1/studio/assets/${encodeURIComponent(assetId)}`
  )
}

export function updateStudioFlowNodes(
  sessionId: string,
  nodes: Array<{
    id: string
    position: { x: number; y: number }
    sort_order: number
  }>
) {
  return apiFetch<void>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/flow`,
    { method: 'PATCH', body: JSON.stringify({ nodes }) }
  )
}

export function createStudioFlowNode(
  sessionId: string,
  input: {
    type: Exclude<StudioFlowNode['type'], 'asset'>
    title: string
    body?: string
    position: { x: number; y: number }
  }
) {
  return apiFetch<StudioFlowNode>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/flow/nodes`,
    { method: 'POST', body: JSON.stringify(input) }
  )
}

export function deleteStudioFlowNode(sessionId: string, nodeId: string) {
  return apiFetch<void>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/flow/nodes/${encodeURIComponent(nodeId)}`,
    { method: 'DELETE' }
  )
}

export function createStudioFlowEdge(
  sessionId: string,
  input: Pick<StudioFlowEdge, 'source' | 'target' | 'label' | 'source_output_key' | 'target_input_key'>
) {
  return apiFetch<StudioFlowEdge>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/flow/edges`,
    { method: 'POST', body: JSON.stringify(input) }
  )
}

export function deleteStudioFlowEdge(sessionId: string, edgeId: string) {
  return apiFetch<void>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/flow/edges/${encodeURIComponent(edgeId)}`,
    { method: 'DELETE' }
  )
}

export type StudioLibraryAssetsQuery = {
  projectId: string
  categoryId?: string
  sessionId?: string
  kind?: string
  format?: string
  rating?: number
  tagIds?: string[]
  sort?: string
  archived?: boolean
  search?: string
  widthMin?: number
  widthMax?: number
  heightMin?: number
  heightMax?: number
  sizeMin?: number
  sizeMax?: number
  addedFrom?: string
  addedTo?: string
  duplicates?: boolean
  cursor?: string
  limit?: number
}

export function studioLibraryDateBoundary(date: string, nextDay = false) {
  const boundary = new Date(`${date}T00:00:00`)
  if (nextDay) boundary.setDate(boundary.getDate() + 1)
  return boundary.toISOString()
}

export function studioLibraryAssetsPath(input: StudioLibraryAssetsQuery) {
  const query = new URLSearchParams({ project_id: input.projectId })
  if (input.search) query.set('q', input.search)
  if (input.kind) query.set('kind', input.kind)
  if (input.format) query.set('format', input.format)
  if (input.categoryId) query.set('category_id', input.categoryId)
  if (input.sessionId) query.set('session_id', input.sessionId)
  if (input.rating !== undefined) query.set('rating', String(input.rating))
  if (input.tagIds?.length) query.set('tag_ids', input.tagIds.join(','))
  if (input.sort) query.set('sort', input.sort)
  if (input.archived !== undefined)
    query.set('archived', String(input.archived))
  if (input.widthMin !== undefined) query.set('width_min', String(input.widthMin))
  if (input.widthMax !== undefined) query.set('width_max', String(input.widthMax))
  if (input.heightMin !== undefined) query.set('height_min', String(input.heightMin))
  if (input.heightMax !== undefined) query.set('height_max', String(input.heightMax))
  if (input.sizeMin !== undefined) query.set('size_min', String(input.sizeMin))
  if (input.sizeMax !== undefined) query.set('size_max', String(input.sizeMax))
  if (input.addedFrom) query.set('added_from', input.addedFrom)
  if (input.addedTo) query.set('added_to', input.addedTo)
  if (input.duplicates) query.set('duplicates', 'true')
  query.set('limit', String(input.limit ?? 50))
  if (input.cursor) query.set('cursor', input.cursor)
  return `/api/v1/studio/library/assets?${query}`
}

export function studioLibraryTreePath(input: {
  projectId: string
  mode: StudioLibraryTreeMode
  parentId?: string
  cursor?: string
}) {
  const query = new URLSearchParams({
    project_id: input.projectId,
    mode: input.mode,
  })
  if (input.parentId) query.set('parent_id', input.parentId)
  if (input.cursor) query.set('cursor', input.cursor)
  return `/api/v1/studio/library/tree?${query}`
}

export function studioLibraryGroupQuery(
  mode: StudioLibraryTreeMode,
  value: string
): Partial<StudioLibraryAssetsQuery> {
  switch (mode) {
    case 'session':
      return { sessionId: value }
    case 'category':
      return { categoryId: value }
    case 'format':
      return { format: value }
    case 'rating':
      return { rating: Number(value) }
    case 'tag':
      return { tagIds: [value] }
    case 'asset':
      return {}
  }
}

export function listStudioLibraryProjects() {
  return apiFetch<StudioLibraryProject[]>('/api/v1/studio/library/projects')
}

export function listStudioLibraryTree(input: {
  projectId: string
  mode: StudioLibraryTreeMode
  parentId?: string
  cursor?: string
}) {
  return apiFetch<StudioLibraryTreePage>(studioLibraryTreePath(input))
}

export function listStudioLibraryAssets(input: StudioLibraryAssetsQuery) {
  return apiFetch<StudioLibraryAssetsPage>(studioLibraryAssetsPath(input))
}

export function getStudioProjectAsset(projectAssetId: string) {
  return apiFetch<StudioProjectAssetDetail>(
    `/api/v1/studio/library/assets/${encodeURIComponent(projectAssetId)}`
  )
}

export function getStudioProjectAssetDuplicates(projectAssetId: string) {
  return apiFetch<{ items: StudioProjectAsset[] }>(
    `/api/v1/studio/library/assets/${encodeURIComponent(projectAssetId)}/duplicates`
  )
}

export function listStudioLibraryFormats(projectId: string) {
  return apiFetch<string[]>(
    `/api/v1/studio/library/formats?project_id=${encodeURIComponent(projectId)}`
  )
}

export function getStudioLibraryPreferences() {
  return apiFetch<StudioLibraryPreferences>(
    '/api/v1/studio/library/preferences'
  )
}

export function updateStudioLibraryPreferences(input: StudioLibraryPreferences) {
  return apiFetch<StudioLibraryPreferences>(
    '/api/v1/studio/library/preferences',
    { method: 'PUT', body: JSON.stringify(input) }
  )
}

export function listStudioLibraryCategories(projectId: string) {
  return apiFetch<StudioLibraryCategory[]>(
    `/api/v1/studio/library/categories?project_id=${encodeURIComponent(projectId)}`
  )
}

export function createStudioLibraryCategory(input: {
  projectId: string
  name: string
  parentId?: string
}) {
  return apiFetch<StudioLibraryCategory>('/api/v1/studio/library/categories', {
    method: 'POST',
    body: JSON.stringify({
      project_id: input.projectId,
      name: input.name,
      parent_id: input.parentId ?? '',
    }),
  })
}

export function updateStudioLibraryCategory(
  categoryId: string,
  input: { name?: string; parentId?: string }
) {
  return apiFetch<StudioLibraryCategory>(
    `/api/v1/studio/library/categories/${encodeURIComponent(categoryId)}`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        ...(input.name !== undefined ? { name: input.name } : {}),
        ...(input.parentId !== undefined
          ? { parent_id: input.parentId }
          : {}),
      }),
    }
  )
}

export function deleteStudioLibraryCategory(categoryId: string) {
  return apiFetch<void>(
    `/api/v1/studio/library/categories/${encodeURIComponent(categoryId)}`,
    { method: 'DELETE' }
  )
}

export function listStudioAssetTags() {
  return apiFetch<StudioAssetTag[]>('/api/v1/studio/library/tags')
}

export function createStudioAssetTag(name: string) {
  return apiFetch<StudioAssetTag>('/api/v1/studio/library/tags', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export function updateStudioProjectAsset(
  projectAssetId: string,
  input: Partial<
    Pick<StudioProjectAsset, 'display_name' | 'category_id' | 'rating'>
  > & { archived?: boolean }
) {
  return apiFetch<StudioProjectAsset>(
    `/api/v1/studio/library/assets/${encodeURIComponent(projectAssetId)}`,
    { method: 'PATCH', body: JSON.stringify(input) }
  )
}

export function updateStudioProjectAssetTags(
  projectAssetId: string,
  tagIds: string[]
) {
  return apiFetch<StudioProjectAsset>(
    `/api/v1/studio/library/assets/${encodeURIComponent(projectAssetId)}/tags`,
    { method: 'PUT', body: JSON.stringify({ tag_ids: tagIds }) }
  )
}

export function updateStudioProjectAssetVersion(
  projectAssetId: string,
  versionId: string
) {
  return apiFetch<StudioProjectAsset>(
    `/api/v1/studio/library/assets/${encodeURIComponent(projectAssetId)}/version`,
    { method: 'PATCH', body: JSON.stringify({ asset_version_id: versionId }) }
  )
}

export function addStudioAssetToProject(
  projectAssetId: string,
  projectId: string,
  versionId: string
) {
  return apiFetch<StudioProjectAsset>(
    `/api/v1/studio/library/assets/${encodeURIComponent(projectAssetId)}/projects`,
    {
      method: 'POST',
      body: JSON.stringify({
        project_id: projectId,
        asset_version_id: versionId,
      }),
    }
  )
}

export type StudioLibraryBatchAction =
  | 'category'
  | 'tags'
  | 'tags_remove'
  | 'rating'
  | 'archive'
  | 'project'

export function updateStudioProjectAssetsBatch(input: {
  projectAssetIds: string[]
  action: StudioLibraryBatchAction
  categoryId?: string
  tagIds?: string[]
  rating?: number
  archived?: boolean
  projectId?: string
}) {
  return apiFetch<{
    results: Array<{ project_asset_id: string; success: boolean; error?: string }>
  }>('/api/v1/studio/library/assets/batch', {
    method: 'POST',
    body: JSON.stringify({
      project_asset_ids: input.projectAssetIds,
      action: input.action,
      ...(input.categoryId !== undefined ? { category_id: input.categoryId } : {}),
      ...(input.tagIds !== undefined ? { tag_ids: input.tagIds } : {}),
      ...(input.rating !== undefined ? { rating: input.rating } : {}),
      ...(input.archived !== undefined ? { archived: input.archived } : {}),
      ...(input.projectId !== undefined ? { project_id: input.projectId } : {}),
    }),
  })
}

export function retryStudioAssetPalette(assetId: string, versionId: string) {
  return apiFetch<void>(
    `/api/v1/studio/assets/${encodeURIComponent(assetId)}/versions/${encodeURIComponent(versionId)}/palette/retry`,
    { method: 'POST' }
  )
}

export async function exportStudioProjectAssets(projectAssetIds: string[]) {
  const token = sessionToken()
  const response = await fetch(`${baseURL()}/api/v1/studio/library/assets/export`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ project_asset_ids: projectAssetIds }),
  })
  if (!response.ok) throw new Error(`导出失败（${response.status}）`)
  return {
    blob: await response.blob(),
    filename: response.headers
      .get('Content-Disposition')
      ?.match(/filename\*=UTF-8''([^;]+)/)?.[1],
  }
}

export function listStudioModels() {
  return apiFetch<StudioModel[]>('/api/v1/studio/models')
}

export type StudioModelConfigInput = {
  name: string
  protocol: StudioModel['protocol']
  baseUrl: string
  model: string
  apiKey: string
  existingModelId?: string
  enabled: boolean
  agentEnabled: boolean
  default: boolean
  limits: StudioModel['limits']
  thinking: StudioModel['thinking']
  capabilities: StudioModel['capabilities']
}

export function createStudioModel(input: StudioModelConfigInput) {
  return apiFetch<StudioModel>('/api/v1/studio/models', {
    method: 'POST',
    body: JSON.stringify({
      name: input.name,
      protocol: input.protocol,
      base_url: input.baseUrl,
      model: input.model,
      api_key: input.apiKey,
      enabled: input.enabled,
      agent_enabled: input.agentEnabled,
      default: input.default,
      limits: input.limits,
      thinking: input.thinking,
      capabilities: input.capabilities,
    }),
  })
}

export function updateStudioModel(
  modelId: string,
  input: StudioModelConfigInput
) {
  return apiFetch<StudioModel>(
    `/api/v1/studio/models/${encodeURIComponent(modelId)}`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        name: input.name,
        protocol: input.protocol,
        base_url: input.baseUrl,
        model: input.model,
        api_key: input.apiKey,
        enabled: input.enabled,
        agent_enabled: input.agentEnabled,
        default: input.default,
        limits: input.limits,
        thinking: input.thinking,
        capabilities: input.capabilities,
      }),
    }
  )
}

export function testStudioModelConfig(input: StudioModelConfigInput) {
  return apiFetch<StudioModelConnectionTest>('/api/v1/studio/models/test', {
    method: 'POST',
    body: JSON.stringify({
      name: input.name,
      protocol: input.protocol,
      base_url: input.baseUrl,
      model: input.model,
      ...(input.apiKey ? { api_key: input.apiKey } : {}),
      ...(input.existingModelId
        ? { existing_model_id: input.existingModelId }
        : {}),
      enabled: input.enabled,
      agent_enabled: input.agentEnabled,
      default: input.default,
      limits: input.limits,
      thinking: input.thinking,
      capabilities: input.capabilities,
    }),
  })
}

export function testStudioModelConnection(modelId: string) {
  return apiFetch<StudioModelConnectionTest>(
    `/api/v1/studio/models/${encodeURIComponent(modelId)}/test`,
    { method: 'POST' }
  )
}

export function listStudioSkills() {
  return apiFetch<StudioSkillSummary[]>('/api/v1/studio/skills')
}

export function getStudioSkill(skillId: string) {
  return apiFetch<StudioSkill>(
    `/api/v1/studio/skills/${encodeURIComponent(skillId)}`
  )
}

export function listStudioSkillVersions(skillId: string) {
  return apiFetch<StudioSkillVersionSummary[]>(
    `/api/v1/studio/skills/${encodeURIComponent(skillId)}/versions`
  )
}

export function getStudioSkillVersion(skillId: string, version: string) {
  return apiFetch<StudioSkillVersion>(
    `/api/v1/studio/skills/${encodeURIComponent(skillId)}/versions/${encodeURIComponent(version)}`
  )
}

export function createStudioSkill(input: {
  name: string
  description: string
  prompt: string
  files?: StudioSkillFile[]
  enabled: boolean
  version?: string
}) {
  return apiFetch<StudioSkill>('/api/v1/studio/skills', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateStudioSkill(input: StudioSkill) {
  return apiFetch<StudioSkill>(
    `/api/v1/studio/skills/${encodeURIComponent(input.id)}`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        name: input.name,
        description: input.description,
        ...(input.files ? { files: input.files } : { prompt: input.prompt }),
        version: input.version,
        updated_at: input.updated_at,
      }),
    }
  )
}

export function inspectStudioSkillZip(file: File) {
  const body = new FormData()
  body.set('file', file)
  return apiFetch<Pick<StudioSkill, 'name' | 'description' | 'files'>>(
    '/api/v1/studio/skills/import/inspect',
    {
      method: 'POST',
      body,
    }
  )
}

export function updateStudioSkillEnabled(skillId: string, enabled: boolean) {
  return apiFetch<StudioSkill>(
    `/api/v1/studio/skills/${encodeURIComponent(skillId)}/enabled`,
    { method: 'PATCH', body: JSON.stringify({ enabled }) }
  )
}

export function listStudioConnectors() {
  return apiFetch<StudioMCPConnector[]>('/api/v1/studio/connectors')
}

export function createStudioConnector(input: {
  name: string
  url: string
  credential: string
  enabled: boolean
  policy: StudioConnectorPolicy
}) {
  return apiFetch<StudioMCPConnector>('/api/v1/studio/connectors', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateStudioConnector(
  input: Pick<
    StudioMCPConnector,
    'id' | 'name' | 'url' | 'enabled' | 'policy'
  > & {
    credential?: string
  }
) {
  return apiFetch<StudioMCPConnector>(
    `/api/v1/studio/connectors/${encodeURIComponent(input.id)}`,
    {
      method: 'PATCH',
      body: JSON.stringify({
        name: input.name,
        url: input.url,
        enabled: input.enabled,
        policy: input.policy,
        ...(input.credential ? { credential: input.credential } : {}),
      }),
    }
  )
}

export function discoverStudioConnector(input: {
  connectorId?: string
  url: string
  credential: string
}) {
  return apiFetch<StudioMCPTool[]>('/api/v1/studio/connectors/discover', {
    method: 'POST',
    body: JSON.stringify({
      connector_id: input.connectorId,
      url: input.url,
      credential: input.credential,
    }),
  })
}

export function probeStudioConnector(connectorId: string) {
  return apiFetch<StudioMCPConnector>(
    `/api/v1/studio/connectors/${encodeURIComponent(connectorId)}/probe`,
    { method: 'POST' }
  )
}

export function listStudioAgentWorkflows() {
  return apiFetch<StudioAgentWorkflow[]>('/api/v1/studio/workflows')
}

export function updateStudioAgentWorkflow(id: string, agentEnabled: boolean) {
  return apiFetch<StudioAgentWorkflow>(
    `/api/v1/studio/workflows/${encodeURIComponent(id)}`,
    { method: 'PATCH', body: JSON.stringify({ agent_enabled: agentEnabled }) }
  )
}

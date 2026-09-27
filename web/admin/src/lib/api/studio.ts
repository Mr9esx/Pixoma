import { apiFetch, toQuery } from './client'

export type StudioPermissionMode =
  | 'request_approval'
  | 'auto_approve'
  | 'full_access'

export type StudioSession = {
  id: string
  title: string
  permission_mode: StudioPermissionMode
  model_config_id?: string
  status: 'active'
  latest_run?: StudioRun | null
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

export type StudioAssetVersion = {
  id: string
  version: number
  mime_type: string
  size_bytes: number
  metadata?: Record<string, unknown>
  content_url: string
  created_at: string
}

export type StudioAsset = {
  id: string
  session_id: string
  name: string
  kind: 'document' | 'image' | 'video' | 'audio' | 'data' | 'file'
  origin: 'user' | 'agent' | 'model' | 'workflow' | 'library'
  source_run_id?: string
  current_version: number
  saved_to_library: boolean
  versions: StudioAssetVersion[]
  created_at: string
  updated_at: string
}

export type StudioLibraryCategory = {
  id: string
  parent_id?: string
  name: string
  created_at: string
  updated_at: string
}

export type StudioLibraryAssetsPage = {
  assets: StudioAsset[]
  total: number
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
  position: { x: number; y: number }
  sort_order: number
  updated_at: string
}

export type StudioFlowEdge = {
  id: string
  source: string
  target: string
  label?: string
}

export type StudioWorkflowExecution = {
  id: string
  run_id: string
  task_id: string
  workflow_id: string
  operation_node_id: string
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
}) {
  return apiFetch<StudioSession[]>(`/api/v1/studio/sessions${toQuery(params)}`)
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
  text: string
  modelConfigId?: string
  permissionMode: StudioPermissionMode
}) {
  return apiFetch<StudioTurn>('/api/v1/studio/messages', {
    method: 'POST',
    body: JSON.stringify({
      session_id: input.sessionId,
      text: input.text,
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

export function resolveStudioApproval(approvalId: string, approved: boolean) {
  return apiFetch<void>(
    `/api/v1/studio/approvals/${encodeURIComponent(approvalId)}`,
    { method: 'POST', body: JSON.stringify({ approved }) }
  )
}

export function saveStudioAssetToLibrary(assetId: string, categoryId?: string) {
  return apiFetch<void>(
    `/api/v1/studio/assets/${encodeURIComponent(assetId)}/save-to-library`,
    { method: 'POST', body: JSON.stringify({ category_id: categoryId ?? '' }) }
  )
}

export function moveStudioLibraryAsset(assetId: string, categoryId?: string) {
  return apiFetch<void>(
    `/api/v1/studio/library/assets/${encodeURIComponent(assetId)}/category`,
    { method: 'PATCH', body: JSON.stringify({ category_id: categoryId ?? '' }) }
  )
}

export function importStudioLibraryAsset(
  sessionId: string,
  assetId: string,
  assetVersionId: string
) {
  return apiFetch<StudioAsset>(
    `/api/v1/studio/sessions/${encodeURIComponent(sessionId)}/assets/import`,
    {
      method: 'POST',
      body: JSON.stringify({
        asset_id: assetId,
        asset_version_id: assetVersionId,
      }),
    }
  )
}

export function createStudioTextAsset(input: {
  sessionId: string
  name: string
  content: string
}) {
  return apiFetch<StudioAsset>('/api/v1/studio/assets/text', {
    method: 'POST',
    body: JSON.stringify({
      session_id: input.sessionId,
      name: input.name,
      content: input.content,
    }),
  })
}

export function updateStudioTextAsset(assetId: string, content: string) {
  return apiFetch<StudioAsset>(
    `/api/v1/studio/assets/${encodeURIComponent(assetId)}/text`,
    { method: 'PATCH', body: JSON.stringify({ content }) }
  )
}

export function getStudioTextAssetContent(contentURL: string) {
  return apiFetch<string>(contentURL)
}

export function uploadStudioAsset(file: File, sessionId?: string) {
  const body = new FormData()
  body.append('file', file)
  if (sessionId) body.append('session_id', sessionId)
  return apiFetch<StudioAsset>('/api/v1/studio/assets/upload', {
    method: 'POST',
    body,
  })
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
  input: Pick<StudioFlowEdge, 'source' | 'target' | 'label'>
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

export function listStudioLibraryAssets(input?: {
  categoryId?: string
  search?: string
  page?: number
  limit?: number
}) {
  const limit = input?.limit ?? 50
  return apiFetch<StudioLibraryAssetsPage>(
    `/api/v1/studio/library/assets${toQuery({
      category_id: input?.categoryId,
      q: input?.search,
      limit,
      offset: ((input?.page ?? 1) - 1) * limit,
    })}`
  )
}

export function listStudioLibraryCategories() {
  return apiFetch<StudioLibraryCategory[]>('/api/v1/studio/library/categories')
}

export function createStudioLibraryCategory(input: {
  name: string
  parentId?: string
}) {
  return apiFetch<StudioLibraryCategory>('/api/v1/studio/library/categories', {
    method: 'POST',
    body: JSON.stringify({ name: input.name, parent_id: input.parentId ?? '' }),
  })
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

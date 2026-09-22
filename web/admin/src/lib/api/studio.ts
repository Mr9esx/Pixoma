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
  created_at: string
  updated_at: string
}

export type StudioMessagePart = {
  type: 'text' | 'image' | 'file' | 'reasoning'
  text?: string
  url?: string
  name?: string
}

export type StudioTranscriptToolCall = {
  id: string
  type: 'function'
  function: { name: string; arguments: string }
}

export type StudioTranscriptMessage = {
  id: string
  role: 'user' | 'assistant' | 'reasoning' | 'tool'
  content: string
  toolCalls?: StudioTranscriptToolCall[]
  toolCallId?: string
  isError?: boolean
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

export type StudioRunEvent = {
  id: string
  run_id: string
  sequence: number
  type: string
  payload: Record<string, unknown>
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

export type StudioLibraryFolder = {
  id: string
  parent_id?: string
  name: string
  created_at: string
  updated_at: string
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

export type StudioSessionDetail = {
  session: StudioSession
  messages: StudioMessage[]
  transcript: StudioTranscript
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

export type StudioSkill = {
  id: string
  name: string
  description: string
  prompt: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export type StudioConnectorPolicy = 'auto' | 'approval' | 'forbidden'

export type StudioMCPConnector = {
  id: string
  name: string
  url: string
  enabled: boolean
  policy: StudioConnectorPolicy
  credential_masked: string
  tools: Array<{ name: string; description: string }>
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
}

export function listStudioSessions(params?: {
  limit?: number
  offset?: number
}) {
  return apiFetch<StudioSession[]>(`/api/v1/studio/sessions${toQuery(params)}`)
}

export function createStudioSession() {
  return apiFetch<StudioSession>('/api/v1/studio/sessions', {
    method: 'POST',
    body: JSON.stringify({}),
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

export function resolveStudioApproval(approvalId: string, approved: boolean) {
  return apiFetch<void>(
    `/api/v1/studio/approvals/${encodeURIComponent(approvalId)}`,
    { method: 'POST', body: JSON.stringify({ approved }) }
  )
}

export function saveStudioAssetToLibrary(assetId: string, folderId?: string) {
  return apiFetch<void>(
    `/api/v1/studio/assets/${encodeURIComponent(assetId)}/save-to-library`,
    { method: 'POST', body: JSON.stringify({ folder_id: folderId ?? '' }) }
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

export function listStudioLibraryAssets(folderId?: string) {
  return apiFetch<StudioAsset[]>(
    `/api/v1/studio/library/assets${toQuery({ folder_id: folderId })}`
  )
}

export function listStudioLibraryFolders() {
  return apiFetch<StudioLibraryFolder[]>('/api/v1/studio/library/folders')
}

export function createStudioLibraryFolder(input: {
  name: string
  parentId?: string
}) {
  return apiFetch<StudioLibraryFolder>('/api/v1/studio/library/folders', {
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
  return apiFetch<StudioSkill[]>('/api/v1/studio/skills')
}

export function createStudioSkill(input: {
  name: string
  description: string
  prompt: string
  enabled: boolean
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
        prompt: input.prompt,
        enabled: input.enabled,
      }),
    }
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
  input: Pick<StudioMCPConnector, 'id' | 'name' | 'url' | 'enabled' | 'policy'>
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
      }),
    }
  )
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

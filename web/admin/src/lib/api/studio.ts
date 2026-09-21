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
  type: 'text' | 'image' | 'file'
  text?: string
  url?: string
  name?: string
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

export type StudioFlowNode = {
  id: string
  type: 'stage' | 'plan' | 'operation' | 'asset'
  title: string
  body?: string
  asset_id?: string
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

export type StudioSkill = {
  id: string
  name: string
  description: string
  prompt: string
  enabled: boolean
  created_at: string
  updated_at: string
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

export function listStudioLibraryAssets(folderId?: string) {
  return apiFetch<StudioAsset[]>(
    `/api/v1/studio/library/assets${toQuery({ folder_id: folderId })}`
  )
}

export function listStudioModels() {
  return apiFetch<StudioModel[]>('/api/v1/studio/models')
}

export function createStudioModel(input: {
  name: string
  protocol: StudioModel['protocol']
  baseUrl: string
  model: string
  apiKey: string
  enabled: boolean
  agentEnabled: boolean
  default: boolean
  thinking: StudioModel['thinking']
  capabilities: StudioModel['capabilities']
}) {
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
      thinking: input.thinking,
      capabilities: input.capabilities,
    }),
  })
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

export type ComfyInstance = {
  id: string
  base_url: string
  enabled: boolean
  capabilities: string[]
  created_at: string
  updated_at: string
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
  id: string
  name: string
  description?: string
  preview?: string
  price: number
  tags?: string[]
  menu_key?: string
  categories?: string[]
  inputs: CaseInputField[]
  outputs: CaseOutputField[]
  bindings: {
    workflow: Record<string, unknown>
    inputs: InputBinding[]
    outputs: OutputBinding[]
  }
  input_schema: Record<string, unknown>
  enabled: boolean
}

export type TaskRecord = {
  id: string
  session_id: string
  chat_id?: number
  case_id: string
  status: string
  instance_id?: string
  prompt_id?: string
  error_code?: string
  error_message?: string
  created_at: string
  updated_at: string
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
  case_id: string
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

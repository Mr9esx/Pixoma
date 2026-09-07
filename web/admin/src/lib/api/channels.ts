import { apiFetch } from './client'

export type Channel = {
  id: string
  platform: string
  name: string
  extra_info?: Record<string, unknown>
  token_masked: string
  enabled: boolean
  created_at: string
  updated_at: string
  adapter_state?: 'absent' | 'starting' | 'running' | 'stopping' | 'error'
  adapter_error?: string
  last_check_kind?: 'ok' | 'network' | 'auth' | 'other'
  last_check_message?: string
  last_check_at?: string
}

export type ChannelReachability = {
  ok: boolean
  kind: 'ok' | 'network' | 'auth' | 'other'
  message: string
  checked_at?: string
}

export function listChannels() {
  return apiFetch<Channel[]>('/api/v1/channels')
}

export function kickChannelProbe() {
  return apiFetch<void>('/api/v1/channels/probe', { method: 'POST' })
}

export function createChannel(body: {
  platform: string
  name: string
  token: string
}) {
  return apiFetch<Channel>('/api/v1/channels', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function getChannel(id: string) {
  return apiFetch<Channel>(`/api/v1/channels/${encodeURIComponent(id)}`)
}

export function updateChannel(
  id: string,
  body: { name?: string; token?: string }
) {
  return apiFetch<Channel>(`/api/v1/channels/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export function setChannelEnabled(id: string, enabled: boolean) {
  return apiFetch<{ enabled: boolean }>(
    `/api/v1/channels/${encodeURIComponent(id)}/${enabled ? 'enable' : 'disable'}`,
    { method: 'POST' }
  )
}

export function deleteChannel(id: string) {
  return apiFetch<{ deleted: boolean }>(
    `/api/v1/channels/${encodeURIComponent(id)}`,
    { method: 'DELETE' }
  )
}

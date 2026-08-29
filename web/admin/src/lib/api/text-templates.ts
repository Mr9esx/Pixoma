import { apiFetch } from './client'

export type TextTemplate = {
  key: string
  description: string
  default: string
  value: string
  variables?: string[]
}

// channelId "" resolves to the platform default; otherwise the per-channel copy.
export function listTextTemplates(channelId = '') {
  const path = channelId
    ? `/api/v1/channels/${encodeURIComponent(channelId)}/text-templates`
    : '/api/v1/text-templates/'
  return apiFetch<TextTemplate[]>(path)
}

export function saveTextTemplates(channelId: string, templates: Record<string, string>) {
  const path = channelId
    ? `/api/v1/channels/${encodeURIComponent(channelId)}/text-templates`
    : '/api/v1/text-templates/'
  return apiFetch<TextTemplate[]>(path, {
    method: 'PUT',
    body: JSON.stringify({ templates }),
  })
}

export function resetTextTemplates(channelId: string, keys: string[]) {
  const path = channelId
    ? `/api/v1/channels/${encodeURIComponent(channelId)}/text-templates/reset`
    : '/api/v1/text-templates/reset'
  return apiFetch<TextTemplate[]>(path, {
    method: 'POST',
    body: JSON.stringify({ keys }),
  })
}

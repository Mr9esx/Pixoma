import type { SessionDraft } from '@/lib/api/types'

export type DraftView = { kind: 'skipped' } | { kind: 'value'; text: string }

export function formatDraftView(entry: SessionDraft): DraftView {
  if (entry.skipped) return { kind: 'skipped' }
  if (entry.text != null && entry.text !== '') {
    return { kind: 'value', text: entry.text }
  }
  if (entry.number != null) return { kind: 'value', text: String(entry.number) }
  if (entry.bool != null) return { kind: 'value', text: String(entry.bool) }
  if (entry.blob != null) {
    return {
      kind: 'value',
      text:
        typeof entry.blob === 'string'
          ? entry.blob
          : JSON.stringify(entry.blob),
    }
  }
  return { kind: 'value', text: '' }
}

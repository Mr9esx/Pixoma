import type { Card } from '@/lib/api/channel-menu'

type NewCardDraftInput = { name: string; text: string }

export function validateNewCardDraft(input: NewCardDraftInput): {
  ok: boolean
  errors: string[]
} {
  const errors: string[] = []
  if (!input.name.trim()) errors.push('name')
  return { ok: errors.length === 0, errors }
}

export function buildNewCardDraft(input: NewCardDraftInput): Card {
  const v = validateNewCardDraft(input)
  if (!v.ok) {
    throw new Error('invalid new card draft: ' + v.errors.join(','))
  }
  return {
    id: `card-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`,
    name: input.name.trim(),
    text: input.text,
    media: [],
    buttons: [],
  }
}

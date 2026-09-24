import type { StudioTrajectoryRecord } from '@/lib/api/studio'

export function systemPromptFromRequest(body: unknown): string | null {
  if (body === null || typeof body !== 'object' || Array.isArray(body))
    return null
  const request = body as Record<string, unknown>
  if (typeof request.system === 'string') return request.system
  const messages = Array.isArray(request.messages)
    ? request.messages
    : Array.isArray(request.input)
      ? request.input
      : []
  const parts: string[] = []
  for (const message of messages) {
    if (
      message === null ||
      typeof message !== 'object' ||
      message.role !== 'system'
    )
      continue
    if (typeof message.content === 'string') parts.push(message.content)
    else if (Array.isArray(message.content)) {
      for (const part of message.content) {
        if (
          part !== null &&
          typeof part === 'object' &&
          typeof part.text === 'string'
        )
          parts.push(part.text)
      }
    }
  }
  return parts.length > 0 ? parts.join('\n\n') : null
}

export function recordPositions(
  records: StudioTrajectoryRecord[],
  actualDuration: boolean
) {
  let minimum = Number.POSITIVE_INFINITY
  let maximum = Number.NEGATIVE_INFINITY
  for (const record of records) {
    const start = Date.parse(record.started_at)
    const end = record.ended_at ? Date.parse(record.ended_at) : start
    if (Number.isFinite(start)) minimum = Math.min(minimum, start)
    if (Number.isFinite(end)) maximum = Math.max(maximum, end)
  }
  if (!Number.isFinite(minimum)) minimum = 0
  if (!Number.isFinite(maximum)) maximum = minimum + 1
  const duration = Math.max(1, maximum - minimum)
  return records.map((record, index) => {
    const start = actualDuration
      ? (Date.parse(record.started_at) - minimum) / duration
      : index / Math.max(1, records.length)
    const end = actualDuration
      ? record.ended_at
        ? (Date.parse(record.ended_at) - minimum) / duration
        : start
      : (index + 1) / Math.max(1, records.length)
    const lane =
      record.kind === 'user'
        ? 0
        : record.kind === 'model' ||
            record.kind === 'assistant' ||
            record.kind === 'reasoning'
          ? 1
          : 2
    return {
      record,
      start: Number.isFinite(start) ? start : 0,
      end: Number.isFinite(end) ? end : start,
      lane,
    }
  })
}

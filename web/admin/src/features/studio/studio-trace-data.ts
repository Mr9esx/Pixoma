import type { StudioTrajectoryRecord } from '@/lib/api/studio'

export function formatDuration(ms: number) {
  return ms < 1000 ? Math.round(ms) + 'ms' : (ms / 1000).toFixed(1) + 's'
}

export function formatRecordTiming(record: StudioTrajectoryRecord) {
  if (record.status === 'running') return '进行中'
  return record.ended_at
    ? formatDuration(Math.max(0, Date.parse(record.ended_at) - Date.parse(record.started_at)))
    : record.status === 'done' ? '已完成' : record.status === 'failed' ? '失败' : record.status
}

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
  const offsets = new Map<number, number>()
  let elapsed = 0
  if (actualDuration) {
    const boundaries = records.flatMap((record) => {
      const start = Date.parse(record.started_at)
      const end = record.ended_at ? Date.parse(record.ended_at) : start
      if (!Number.isFinite(start) || !Number.isFinite(end) || end < start)
        throw new RangeError(`轨迹记录时间无效：${record.id}`)
      return [{ time: start, change: 1 }, { time: end, change: -1 }]
    }).sort((a, b) => a.time - b.time)
    let active = 0
    let previous = boundaries[0]?.time ?? 0
    for (const boundary of boundaries) {
      if (active > 0) elapsed += boundary.time - previous
      offsets.set(boundary.time, elapsed)
      active += boundary.change
      previous = boundary.time
    }
    if (elapsed === 0 && boundaries.length > 0) {
      const first = boundaries[0].time
      for (const boundary of boundaries) offsets.set(boundary.time, boundary.time - first)
      elapsed = boundaries[boundaries.length - 1].time - first
    }
  }
  const duration = Math.max(1, elapsed)
  return records.map((record, index) => {
    const start = actualDuration
      ? offsets.get(Date.parse(record.started_at))! / duration
      : index / Math.max(1, records.length)
    const end = actualDuration
      ? record.ended_at
        ? offsets.get(Date.parse(record.ended_at))! / duration
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
      start,
      end,
      lane,
    }
  })
}

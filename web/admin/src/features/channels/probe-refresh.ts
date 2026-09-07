export type ProbeStamp = {
  id: string
  enabled: boolean
  last_check_at?: string
}

export function captureProbeStamps(
  items: ProbeStamp[]
): Map<string, string | undefined> {
  return new Map(
    items.filter((ch) => ch.enabled).map((ch) => [ch.id, ch.last_check_at])
  )
}

export function probeHasSettled(
  before: ReadonlyMap<string, string | undefined>,
  after: ProbeStamp[]
): boolean {
  const enabled = after.filter((ch) => ch.enabled)
  if (enabled.length === 0) return true
  return enabled.every((ch) => {
    const prev = before.get(ch.id)
    return Boolean(ch.last_check_at) && ch.last_check_at !== prev
  })
}

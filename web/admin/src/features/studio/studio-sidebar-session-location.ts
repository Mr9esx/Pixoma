export type StudioSessionListSource = 'project' | 'recent'

export type StudioSessionListSelection = {
  sessionId: string
  source: StudioSessionListSource
}

export function isActiveStudioSessionItem({
  activeSessionId,
  selection,
  sessionId,
  sessionProjectId,
  source,
}: {
  activeSessionId?: string
  selection?: StudioSessionListSelection
  sessionId: string
  sessionProjectId?: string
  source: StudioSessionListSource
}): boolean {
  if (activeSessionId !== sessionId) return false

  const selectedSource =
    selection?.sessionId === sessionId &&
    (selection.source === 'recent' || Boolean(sessionProjectId))
      ? selection.source
      : undefined
  const activeSource =
    selectedSource ?? (sessionProjectId ? 'project' : 'recent')

  return activeSource === source
}

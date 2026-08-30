export const SESSION_KEY = 'pixoma:quick-config'
export const SESSION_SCHEMA_VERSION = 3

export type TopicDraft = { key: string; name: string }

export type QuickConfigSession = {
  caseId: number | null
  step: number
  caseDraft: unknown
  topicKey: string | null
  topicDraft: TopicDraft | null
  selectedEdgeId: string | null
  schemaVersion: typeof SESSION_SCHEMA_VERSION
  updatedAt: string
}

type StorageLike = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

export function saveQuickConfigSession(
  storage: StorageLike,
  session: Omit<QuickConfigSession, 'schemaVersion'>
): void {
  storage.setItem(
    SESSION_KEY,
    JSON.stringify({ ...session, schemaVersion: SESSION_SCHEMA_VERSION })
  )
}

export function loadQuickConfigSession(
  storage: StorageLike
): QuickConfigSession | null {
  const raw = storage.getItem(SESSION_KEY)
  if (!raw) return null
  try {
    const parsed: unknown = JSON.parse(raw)
    if (typeof parsed !== 'object' || parsed === null) return null
    const candidate = parsed as Partial<QuickConfigSession>
    if (candidate.schemaVersion !== SESSION_SCHEMA_VERSION) {
      storage.removeItem(SESSION_KEY)
      return null
    }
    if (
      (candidate.caseId !== null &&
        typeof candidate.caseId !== 'number') ||
      typeof candidate.step !== 'number' ||
      typeof candidate.updatedAt !== 'string'
    ) {
      return null
    }
    return {
      caseId: candidate.caseId ?? null,
      step: candidate.step,
      caseDraft: candidate.caseDraft ?? null,
      topicKey:
        typeof candidate.topicKey === 'string' ? candidate.topicKey : null,
      topicDraft:
        candidate.topicDraft &&
        typeof candidate.topicDraft === 'object' &&
        typeof candidate.topicDraft.key === 'string'
          ? candidate.topicDraft
          : null,
      selectedEdgeId:
        typeof candidate.selectedEdgeId === 'string'
          ? candidate.selectedEdgeId
          : null,
      schemaVersion: candidate.schemaVersion,
      updatedAt: candidate.updatedAt,
    }
  } catch {
    return null
  }
}

export function clearQuickConfigSession(storage: StorageLike): void {
  storage.removeItem(SESSION_KEY)
}

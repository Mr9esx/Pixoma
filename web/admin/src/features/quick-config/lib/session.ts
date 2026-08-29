export const SESSION_KEY = 'pixoma:quick-config'
export const SESSION_SCHEMA_VERSION = 2

export type PendingMenuEntry = {
  channelId: string
  label: string
  mode: 'direct' | 'list'
}

export type QuickConfigSession = {
  caseId: number | null
  mode: 'create' | 'existing'
  step: number
  /** 未提交的 Case 草稿（完成页统一落库）。 */
  caseDraft: unknown
  routing: unknown
  pendingEntries: PendingMenuEntry[]
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
      (candidate.mode !== 'create' && candidate.mode !== 'existing') ||
      typeof candidate.step !== 'number' ||
      typeof candidate.updatedAt !== 'string'
    ) {
      return null
    }
    return {
      caseId: candidate.caseId ?? null,
      mode: candidate.mode,
      step: candidate.step,
      caseDraft: candidate.caseDraft ?? null,
      routing: candidate.routing,
      pendingEntries: Array.isArray(candidate.pendingEntries)
        ? (candidate.pendingEntries as PendingMenuEntry[])
        : [],
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

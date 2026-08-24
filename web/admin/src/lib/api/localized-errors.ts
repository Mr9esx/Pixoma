import { ApiError } from './client'

/** Backend delete-conflict error codes → i18n keys. */
const CASE_DELETE_ERROR_CODES: Record<string, string> = {
  case_delete_needs_ack: 'cases.deleteNeedsAck',
}

/** Translate a case-delete error into localized guidance, if recognised. */
export function caseDeleteErrorMessage(
  err: unknown,
  t: (key: string) => string
): string | undefined {
  if (!(err instanceof ApiError)) return undefined
  const key = err.code ? CASE_DELETE_ERROR_CODES[err.code] : undefined
  return key ? t(key) : undefined
}

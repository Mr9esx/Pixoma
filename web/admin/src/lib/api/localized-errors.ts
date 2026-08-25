import { ApiError } from './client'

/** Backend delete-conflict error codes → i18n keys. */
const CASE_DELETE_ERROR_CODES: Record<string, string> = {
  case_delete_needs_ack: 'cases.deleteNeedsAck',
}

const TOPIC_DELETE_ERROR_CODES: Record<string, string> = {
  topic_delete_needs_ack: 'topics.deleteNeedsAck',
  topic_default_protected: 'topics.deleteDefaultProtected',
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

/** Translate a topic-delete error into localized guidance, if recognised. */
export function topicDeleteErrorMessage(
  err: unknown,
  t: (key: string) => string
): string | undefined {
  if (!(err instanceof ApiError)) return undefined
  const key = err.code ? TOPIC_DELETE_ERROR_CODES[err.code] : undefined
  return key ? t(key) : undefined
}

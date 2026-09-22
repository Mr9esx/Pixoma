import { ApiError } from './client'

/**
 * 删除冲突的业务错误码 → i18n 词条。
 *
 * 后端返回 409 时，前端把「同时清理引用」这个交互选项讲清楚。
 */
const CASE_DELETE_ERROR_CODES: Record<number, string> = {
  4090604: 'cases.deleteNeedsAck',
}

const TOPIC_DELETE_ERROR_CODES: Record<number, string> = {
  4090904: 'topics.deleteNeedsAck',
  4090913: 'topics.deleteDefaultProtected',
}

/** Translate a case-delete error into localized guidance, if recognised. */
export function caseDeleteErrorMessage(
  err: unknown,
  t: (key: string) => string
): string | undefined {
  if (!(err instanceof ApiError)) return undefined
  const key = CASE_DELETE_ERROR_CODES[err.code]
  return key ? t(key) : undefined
}

/** Translate a topic-delete error into localized guidance, if recognised. */
export function topicDeleteErrorMessage(
  err: unknown,
  t: (key: string) => string
): string | undefined {
  if (!(err instanceof ApiError)) return undefined
  const key = TOPIC_DELETE_ERROR_CODES[err.code]
  return key ? t(key) : undefined
}

import { ApiError } from './client'
import { apiErrorMessage } from './error-copy'

/** Map cancel / detail API errors to i18n keys or backend message. */
export function taskActionErrorMessage(
  err: unknown,
  t: (key: string) => string,
): string {
  if (err instanceof ApiError) {
    if (err.status === 409) return t('tasks.cannotCancel')
    if (err.status === 404) return t('tasks.notFound')
  }
  return apiErrorMessage(err, t)
}

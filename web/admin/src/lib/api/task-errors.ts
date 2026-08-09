import { ApiError } from './client'

/** Map cancel / detail API errors to i18n keys or backend message. */
export function taskActionErrorMessage(
  err: unknown,
  t: (key: string) => string,
): string {
  if (err instanceof ApiError) {
    if (err.status === 409) return t('tasks.cannotCancel')
    if (err.status === 404) return t('tasks.notFound')
    if (err.message) return err.message
  }
  if (err instanceof Error && err.message) return err.message
  return t('common.errorGeneric')
}

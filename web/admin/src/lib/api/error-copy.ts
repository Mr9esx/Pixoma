import { ApiError } from './client'

type Translate = (key: string) => string

/**
 * 把错误翻成当前语言的用户文案。
 *
 * 优先用业务错误码查 `apiError.<code>` 词条；没有对应词条时退回后端文案，
 * 再退回通用提示。i18next 找不到词条会把 key 原样返回，所以要比对一次。
 */
export function apiErrorMessage(err: unknown, t: Translate): string {
  if (err instanceof ApiError) {
    const key = `apiError.${err.code}`
    if (err.code) {
      const localized = t(key)
      if (localized && localized !== key) return localized
    }
    if (err.message) return err.message
  }
  if (err instanceof Error && err.message) return err.message
  return t('common.errorGeneric')
}

/** 取出后端给出的原始技术原因，没有则返回空串。 */
export function apiErrorDetail(err: unknown): string {
  return err instanceof ApiError ? (err.detail ?? '') : ''
}

/**
 * 拼出「用户文案 + 原始技术原因」两行文本，用于 toast。
 *
 * 技术原因是排障线索，直接展示，不做翻译。
 */
export function apiErrorText(err: unknown, t: Translate): string {
  const message = apiErrorMessage(err, t)
  const detail = apiErrorDetail(err)
  return detail && detail !== message ? `${message}\n${detail}` : message
}

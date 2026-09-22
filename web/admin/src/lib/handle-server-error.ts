import { AxiosError } from 'axios'
import { toast } from 'sonner'
import { i18n } from './i18n'
import { apiErrorText } from './api/error-copy'

export function handleServerError(error: unknown) {
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.log(error)
  }

  if (error instanceof AxiosError) {
    const title = error.response?.data?.title
    if (typeof title === 'string' && title.length > 0) {
      toast.error(title)
      return
    }
  }

  toast.error(apiErrorText(error, (key) => i18n.t(key)))
}

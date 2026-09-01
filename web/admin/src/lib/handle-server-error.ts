import { AxiosError } from 'axios'
import { toast } from 'sonner'
import { ApiError } from './api/client'

export function handleServerError(error: unknown) {
  if (import.meta.env.DEV) {
    // eslint-disable-next-line no-console
    console.log(error)
  }

  let errMsg = 'Something went wrong!'

  if (error instanceof ApiError) {
    errMsg = error.message
  }

  if (
    error &&
    typeof error === 'object' &&
    'status' in error &&
    Number(error.status) === 204
  ) {
    errMsg = 'No content.'
  }

  if (error instanceof AxiosError) {
    const title = error.response?.data?.title
    if (typeof title === 'string' && title.length > 0) {
      errMsg = title
    }
  }

  toast.error(errMsg)
}

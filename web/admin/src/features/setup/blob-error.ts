import { apiErrorDetail } from '../../lib/api/error-copy'
import type { AlertCopy } from './db-error'

const BLOB_ERROR_PATTERNS: Array<{ pattern: RegExp; key: string }> = [
  {
    pattern: /InvalidAccessKeyId|SignatureDoesNotMatch|AccessDenied/i,
    key: 'setup.errBlobKey',
  },
  {
    pattern: /NoSuchBucket|NoSuchKey/i,
    key: 'setup.errBlobBucket',
  },
  {
    pattern: /connection refused|i\/o timeout|timeout|no such host|unreachable/i,
    key: 'setup.errBlobNet',
  },
  {
    pattern: /empty (endpoint|region|bucket)|missing/i,
    key: 'setup.errBlobIncomplete',
  },
]

export function blobErrorCopy(err: unknown): AlertCopy {
  // 与 setupErrorCopy 一致：模式匹配原始技术原因。
  const detail = apiErrorDetail(err) || (err instanceof Error ? err.message : String(err))
  const matched = BLOB_ERROR_PATTERNS.find(({ pattern }) => pattern.test(detail))
  return {
    key: matched?.key ?? 'setup.errBlobGeneric',
    detail,
  }
}
